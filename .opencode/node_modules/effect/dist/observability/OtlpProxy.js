/**
 * Proxies OTLP HTTP exports from browsers through an application's own HTTP
 * server to an OpenTelemetry Collector or vendor endpoint.
 *
 * A browser exporting OTLP directly would need the vendor credentials in its
 * bundle and a cross-origin collector. `OtlpProxy` mounts `POST /v1/traces`,
 * `/v1/logs`, and `/v1/metrics` on an `HttpRouter`, adds the configured
 * headers on the server, and forwards the payload unchanged. The browser can
 * then use `Otlp.layer({ baseUrl: "/otlp" })` against its own origin.
 *
 * **Details**
 *
 * - The upstream URL comes only from configuration. The request path and query
 *   string never reach it.
 * - Only `application/json` and `application/x-protobuf` bodies are accepted;
 *   other content types get `415`. Payloads are not decoded or validated.
 * - Only `content-type` and `content-encoding` are forwarded from the client.
 *   Configured `headers` are applied last, so a client cannot override them.
 * - Bodies are buffered up to `maxBodySize` and rejected with `413` beyond it.
 * - Upstream requests send `accept-encoding: identity`, and the client's
 *   `accept-encoding` is never forwarded.
 * - The upstream status and body are returned with only the `content-type` and
 *   `retry-after` headers, so client SDK back-off keeps working and upstream
 *   cookies are dropped. Transport failures return `502` and timeouts `504`.
 *   The proxy does not retry or batch.
 *
 * **Gotchas**
 *
 * - Anyone who can reach the routes can send telemetry with the configured
 *   credentials. Compose authentication, CORS, and rate limiting around the
 *   layer as the application requires.
 * - The response body is returned as the `HttpClient` provides it, without a
 *   `content-encoding` header. Setting `accept-encoding` in `headers` can
 *   invalidate this, as can an upstream that compresses despite
 *   `accept-encoding: identity`: clients that do not decode responses, such as
 *   `NodeHttpClient.makeUndici`, then return compressed bytes.
 * - Requests to the proxy routes are traced by the HTTP server like any other
 *   route. To suppress those spans, provide `HttpMiddleware.TracerDisabledWhen`
 *   with a predicate matching the routes, or use
 *   `HttpMiddleware.layerTracerDisabledForUrls`, which matches exact URLs, so
 *   list each route.
 *
 * @stability unstable
 * @since 4.0.7
 */
import * as ByteSize from "../ByteSize.js";
import * as Duration from "../Duration.js";
import * as Effect from "../Effect.js";
import { constTrue } from "../Function.js";
import * as Headers from "../http/Headers.js";
import * as HttpClient from "../http/HttpClient.js";
import * as HttpClientRequest from "../http/HttpClientRequest.js";
import * as HttpIncomingMessage from "../http/HttpIncomingMessage.js";
import * as HttpRouter from "../http/HttpRouter.js";
import * as HttpServerRequest from "../http/HttpServerRequest.js";
import * as HttpServerRespondable from "../http/HttpServerRespondable.js";
import * as HttpServerResponse from "../http/HttpServerResponse.js";
import * as Layer from "../Layer.js";
import * as OtlpEnv from "./internal/otlpEnv.js";
/**
 * Creates a record of request handlers, one for each OTLP signal, that forward
 * to the configured upstream.
 *
 * **When to use**
 *
 * Use to mount the proxy on routes or a router other than the ones `layer`
 * registers, such as an `HttpApi` endpoint.
 *
 * **Details**
 *
 * Each signal is sent to `urls[signal]` when set, otherwise to
 * `/v1/<signal>` below `baseUrl`. `maxBodySize` defaults to the
 * `HttpIncomingMessage.MaxBodySize` reference, then 10 MiB. `timeout` defaults
 * to 10 seconds.
 *
 * @see {@link layer} for mounting the handlers on an `HttpRouter`.
 * @stability unstable
 * @category constructors
 * @since 4.0.7
 */
export const make = options => Effect.gen(function* () {
  const client = yield* HttpClient.HttpClient;
  const base = HttpClientRequest.post(options.baseUrl);
  const handlers = {};
  for (const signal of allSignals) {
    handlers[signal] = makeHandler(client, {
      ...options,
      url: options.urls?.[signal] ?? HttpClientRequest.appendUrl(base, `/v1/${signal}`).url
    });
  }
  return handlers;
});
/**
 * Creates a layer that mounts the OTLP proxy routes on an `HttpRouter`.
 *
 * **Details**
 *
 * Registers `POST <prefix>/v1/<signal>` for each of `signals`, which defaults
 * to traces, logs, and metrics. Without a `prefix` the routes are mounted at
 * the router root.
 *
 * **Example** (Proxying browser telemetry to a vendor)
 *
 * ```ts import.meta.vitest
 * import { Layer } from "effect"
 * import { FetchHttpClient } from "effect/http"
 * import { OtlpProxy } from "effect/observability"
 *
 * const OtlpProxyLayer = OtlpProxy.layer({
 *   baseUrl: "https://otlp.vendor.example",
 *   headers: { authorization: "Bearer server-side-key" },
 *   prefix: "/otlp"
 * }).pipe(Layer.provide(FetchHttpClient.layer))
 *
 * Layer.isLayer(OtlpProxyLayer) // => true
 * ```
 *
 * @see {@link layerFromConfig} for reading the upstream from OpenTelemetry environment variables.
 * @stability unstable
 * @category layers
 * @since 4.0.7
 */
export const layer = options => Layer.effectDiscard(Effect.gen(function* () {
  const handlers = yield* make(options);
  yield* addRoutes(options.prefix, (options.signals ?? allSignals).map(signal => [signal, handlers[signal]]));
}));
/**
 * Creates a layer that mounts the OTLP proxy routes using the upstream from
 * OpenTelemetry environment configuration.
 *
 * **Details**
 *
 * Each signal is sent to `OTEL_EXPORTER_OTLP_<SIGNAL>_ENDPOINT`, falling back
 * to `/v1/<signal>` below `OTEL_EXPORTER_OTLP_ENDPOINT`, with headers from
 * `OTEL_EXPORTER_OTLP_<SIGNAL>_HEADERS` or `OTEL_EXPORTER_OTLP_HEADERS`. The
 * `headers` option replaces the configured headers. Signals without an
 * endpoint get no route.
 *
 * @see {@link layer} for configuring the upstream directly.
 * @stability unstable
 * @category layers
 * @since 4.0.7
 */
export const layerFromConfig = options => Layer.effectDiscard(Effect.gen(function* () {
  const client = yield* HttpClient.HttpClient;
  const routes = [];
  for (const signal of options?.signals ?? allSignals) {
    const envSignal = signal.toUpperCase();
    const url = yield* OtlpEnv.endpoint(envSignal);
    if (url === undefined) continue;
    routes.push([signal, makeHandler(client, {
      ...options,
      url: url.toString(),
      headers: options?.headers ?? (yield* OtlpEnv.headers(envSignal))
    })]);
  }
  yield* addRoutes(options?.prefix, routes);
}).pipe(Effect.orDie));
// internal
const allSignals = ["traces", "logs", "metrics"];
const contentTypes = /*#__PURE__*/new Set(["application/json", "application/x-protobuf"]);
const defaultMaxBodySize = /*#__PURE__*/ByteSize.mebibytes(10);
const addRoutes = /*#__PURE__*/Effect.fnUntraced(function* (prefix, routes) {
  const router = yield* HttpRouter.HttpRouter;
  const target = prefix === undefined ? router : router.prefixed(prefix);
  for (const [signal, handler] of routes) {
    yield* target.add("POST", `/v1/${signal}`, handler);
  }
});
const makeHandler = (client, options) => {
  // Skips the outbound span and trace headers in one step.
  const upstream = client.pipe(HttpClient.transformResponse(Effect.provideService(HttpClient.TracerDisabledWhen, constTrue)));
  const headers = Headers.fromInput(options.headers);
  const maxBodySize = options.maxBodySize === undefined ? undefined : ByteSize.fromInputUnsafe(options.maxBodySize);
  const timeout = options.timeout ?? Duration.seconds(10);
  return Effect.gen(function* () {
    const request = yield* HttpServerRequest.HttpServerRequest;
    const contentType = request.headers["content-type"];
    if (!isSupportedContentType(contentType)) {
      return unsupportedMediaType;
    }
    const limit = maxBodySize ?? (yield* HttpIncomingMessage.MaxBodySize) ?? defaultMaxBodySize;
    // Node's reader only counts bytes as they arrive, and would send
    // 100 Continue first, so reject an over-limit declared length up front.
    const contentLength = parseContentLength(request.headers["content-length"]);
    if (contentLength !== undefined && contentLength > Number(limit)) {
      return contentTooLarge;
    }
    const body = yield* request.arrayBuffer.pipe(Effect.provideService(HttpIncomingMessage.MaxBodySize, limit), Effect.catch(HttpServerRespondable.toResponse));
    if (HttpServerResponse.isHttpServerResponse(body)) {
      return body;
    }
    let outbound = HttpClientRequest.bodyUint8Array(HttpClientRequest.post(options.url), new Uint8Array(body), contentType);
    const contentEncoding = request.headers["content-encoding"];
    if (contentEncoding !== undefined) {
      outbound = HttpClientRequest.setHeader(outbound, "content-encoding", contentEncoding);
    }
    // The response is returned without content-encoding, so request it
    // uncompressed. Configured headers can still override this.
    outbound = HttpClientRequest.setHeader(outbound, "accept-encoding", "identity");
    return yield* upstream.execute(HttpClientRequest.setHeaders(outbound, headers)).pipe(Effect.flatMap(response => Effect.map(response.arrayBuffer, body => toServerResponse(response, body))), Effect.timeout(timeout), Effect.catchTags({
      TimeoutError: () => Effect.succeed(gatewayTimeout),
      HttpClientError: () => Effect.succeed(badGateway)
    }));
  });
};
const unsupportedMediaType = /*#__PURE__*/HttpServerResponse.empty({
  status: 415
});
const contentTooLarge = /*#__PURE__*/HttpServerResponse.empty({
  status: 413
});
const badGateway = /*#__PURE__*/HttpServerResponse.empty({
  status: 502
});
const gatewayTimeout = /*#__PURE__*/HttpServerResponse.empty({
  status: 504
});
const isSupportedContentType = contentType => contentType !== undefined && (contentTypes.has(contentType) || contentTypes.has(contentType.split(";")[0].trim().toLowerCase()));
const parseContentLength = contentLength => contentLength !== undefined && /^\d+$/.test(contentLength) ? Number(contentLength) : undefined;
const toServerResponse = (response, body) => {
  const headers = {};
  if (response.headers["retry-after"] !== undefined) {
    headers["retry-after"] = response.headers["retry-after"];
  }
  return body.byteLength === 0 ? HttpServerResponse.empty({
    status: response.status,
    headers
  }) : HttpServerResponse.uint8Array(new Uint8Array(body), {
    status: response.status,
    headers,
    contentType: response.headers["content-type"]
  });
};
//# sourceMappingURL=OtlpProxy.js.map