/**
 * Fetch-based implementation of the Effect HTTP client service.
 *
 * This module provides an `HttpClient` layer that executes requests through a
 * Web Fetch API implementation. It is the transport to use in browsers, edge
 * runtimes, and Node.js environments where `globalThis.fetch` is available, or
 * anywhere a compatible fetch function can be supplied.
 *
 * Requests with an `Upgrade` header fail with `TransportError` before anything
 * is sent because fetch cannot hand over the connection.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Context from "../Context.js";
import * as Effect from "../Effect.js";
import * as Stream from "../Stream.js";
import * as Headers from "./Headers.js";
import * as HttpClient from "./HttpClient.js";
import * as HttpClientError from "./HttpClientError.js";
import * as HttpClientResponse from "./HttpClientResponse.js";
/**
 * Context reference for the `fetch` implementation used by the fetch-based HTTP client.
 *
 * **Details**
 *
 * Defaults to `globalThis.fetch`.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export const Fetch = /*#__PURE__*/Context.Reference("effect/http/FetchHttpClient/Fetch", {
  defaultValue: () => globalThis.fetch
});
/**
 * Context reference controlling stream bodies when the runtime does not support streaming requests.
 *
 * **Details**
 *
 * Defaults to `"error"`, which fails with an `EncodeError` without sending the
 * request. Set to `"buffer"` to collect the entire stream in memory and send it
 * as a single body instead. Runtimes with streaming support always send the stream.
 *
 * **Gotchas**
 *
 * Buffering keeps the entire upload in memory and can exhaust memory for large
 * or unbounded uploads. Support is detected once, on the first stream body sent,
 * using the global `Request` implementation even when a custom `Fetch` is provided.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export const StreamBodyFallback = /*#__PURE__*/Context.Reference("effect/http/FetchHttpClient/StreamBodyFallback", {
  defaultValue: () => "error"
});
/**
 * Service that contains default fetch options for the fetch-based HTTP client.
 *
 * **When to use**
 *
 * Use to provide default credentials, cache, redirect, integrity, or other
 * fetch options for outgoing HTTP requests.
 *
 * **Details**
 *
 * Request-specific method, headers, body, and abort signal are supplied by the client when a request is executed.
 *
 * @see {@link HttpClient.followRedirects} for following redirects in the client, which needs `{ redirect: "manual" }` here
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export class RequestInit extends /*#__PURE__*/Context.Service()("effect/http/FetchHttpClient/RequestInit") {}
// https://developer.chrome.com/docs/capabilities/web-apis/fetch-streaming-requests
let supportsStreamBody;
const detectStreamBodySupport = () => {
  try {
    let duplexAccessed = false;
    const body = new ReadableStream();
    const request = new Request("https://example.com", {
      method: "POST",
      body,
      get duplex() {
        duplexAccessed = true;
        return "half";
      }
    });
    // Bun and Deno preserve the stream without reading the duplex option.
    return (duplexAccessed || request.body === body) && !request.headers.has("Content-Type");
  } catch {
    return false;
  }
};
const fetch = /*#__PURE__*/HttpClient.make((request, url, signal, fiber) => {
  if (request.headers.upgrade !== undefined) {
    return Effect.fail(new HttpClientError.HttpClientError({
      reason: new HttpClientError.TransportError({
        request,
        description: "fetch cannot send an HTTP upgrade request"
      })
    }));
  }
  const fetch = fiber.getRef(Fetch);
  const options = Context.getOrUndefined(fiber.context, RequestInit) ?? {};
  let headers = options.headers ? Headers.merge(Headers.fromInput(options.headers), request.headers) : request.headers;
  if (headers["content-length"]) {
    headers = Headers.remove(headers, "content-length");
  }
  let uploadController;
  const stopUpload = Effect.sync(() => uploadController?.abort());
  const send = body => Effect.map(Effect.tryPromise({
    try: () => fetch(url, {
      ...options,
      method: request.method,
      headers,
      body,
      duplex: typeof ReadableStream !== "undefined" && body instanceof ReadableStream ? "half" : undefined,
      signal: uploadController ? AbortSignal.any([signal, uploadController.signal]) : signal
    }),
    catch: cause => new HttpClientError.HttpClientError({
      reason: new HttpClientError.TransportError({
        request,
        cause
      })
    })
  }), response => {
    const original = HttpClientResponse.fromWeb(request, response);
    if (!uploadController) return original;
    if (response.body === null) uploadController.abort();
    // Finishing or releasing either response reader ends the upload, not
    // merely receiving headers. Keep the original response and its caches,
    // and inherit its Web source so `toWeb` hands over the original body.
    return Object.create(original, {
      ...Object.fromEntries(["arrayBuffer", "text", "json", "formData", "urlParamsBody"].map(key => [key, {
        get: () => Effect.ensuring(original[key], stopUpload)
      }])),
      stream: {
        get: () => Stream.ensuring(original.stream, stopUpload)
      }
    });
  }).pipe(Effect.onError(() => stopUpload));
  switch (request.body._tag) {
    case "Raw":
    case "Uint8Array":
      return send(request.body.body);
    case "FormData":
      return send(request.body.formData);
    case "Stream":
      {
        supportsStreamBody ??= detectStreamBodySupport();
        if (supportsStreamBody) {
          uploadController = new AbortController();
          const uploadSignal = AbortSignal.any([signal, uploadController.signal]);
          const stopped = Effect.callback(resume => {
            const onAbort = () => resume(Effect.interrupt);
            if (uploadSignal.aborted) {
              onAbort();
              return;
            }
            uploadSignal.addEventListener("abort", onAbort, {
              once: true
            });
            return Effect.sync(() => uploadSignal.removeEventListener("abort", onAbort));
          });
          return Effect.flatMap(Stream.toReadableStreamEffect(Stream.interruptWhen(request.body.stream, stopped)), send);
        }
        if (fiber.getRef(StreamBodyFallback) === "error") {
          return Effect.fail(new HttpClientError.HttpClientError({
            reason: new HttpClientError.EncodeError({
              request,
              description: "Streaming request bodies are not supported in this runtime"
            })
          }));
        }
        return Stream.mkArrayBuffer(request.body.stream).pipe(Effect.mapError(cause => new HttpClientError.HttpClientError({
          reason: new HttpClientError.EncodeError({
            request,
            cause
          })
        })), Effect.flatMap(send));
      }
  }
  return send(undefined);
});
/**
 * Layer that provides an `HttpClient` implementation backed by the configured
 * `Fetch` function.
 *
 * **When to use**
 *
 * Use when an Effect program should execute `HttpClient` requests through the
 * platform `fetch` implementation, especially in browser, edge, or Node.js
 * runtimes with `globalThis.fetch`.
 *
 * **Details**
 *
 * The layer uses the current `Fetch` reference and optional `RequestInit`
 * service for each request. Request-specific method, headers, body, and abort
 * signal are supplied by the client and override matching `RequestInit` fields.
 *
 * **Gotchas**
 *
 * Fetch behavior comes from the runtime's implementation, so CORS, cookies,
 * redirects, abort handling, and streaming support can vary by platform.
 * `fetch` follows redirects itself by default, so `HttpClient.followRedirects`
 * has no effect unless `RequestInit` sets `redirect: "manual"`. Trace
 * propagation is on by default, and a browser preflight fails unless the
 * target allows `b3` and `traceparent`. Stream request bodies are sent as Web
 * streams with `duplex: "half"` when the runtime supports it; see
 * `StreamBodyFallback` for the unsupported case. Any `content-length` header is
 * removed before calling `fetch`. Chrome passes the support check but rejects
 * streaming uploads over HTTP/1.1, which surfaces as a `TransportError`, and
 * browsers only deliver the response after the entire request body has been sent.
 *
 * @see {@link Fetch} for supplying the fetch implementation used by this layer
 * @see {@link RequestInit} for default `RequestInit` options applied before request-specific fields
 * @see {@link StreamBodyFallback} for opting into buffering on unsupported runtimes
 * @see {@link HttpClient.TracerPropagationEnabled} for leaving `b3` and `traceparent` off a browser request
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layer = /*#__PURE__*/HttpClient.layerMergedContext(/*#__PURE__*/Effect.succeed(fetch));
//# sourceMappingURL=FetchHttpClient.js.map