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
import * as Context from "../Context.ts";
import type * as Layer from "../Layer.ts";
import * as HttpClient from "./HttpClient.ts";
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
export declare const Fetch: Context.Reference<typeof globalThis.fetch>;
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
export declare const StreamBodyFallback: Context.Reference<"buffer" | "error">;
declare const RequestInit_base: Context.ServiceClass<RequestInit, "effect/http/FetchHttpClient/RequestInit", globalThis.RequestInit>;
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
export declare class RequestInit extends RequestInit_base {
}
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
export declare const layer: Layer.Layer<HttpClient.HttpClient>;
export {};
//# sourceMappingURL=FetchHttpClient.d.ts.map