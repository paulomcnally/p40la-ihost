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
import * as ByteSize from "../ByteSize.ts";
import * as Duration from "../Duration.ts";
import * as Effect from "../Effect.ts";
import * as Headers from "../http/Headers.ts";
import * as HttpClient from "../http/HttpClient.ts";
import * as HttpRouter from "../http/HttpRouter.ts";
import * as HttpServerRequest from "../http/HttpServerRequest.ts";
import * as HttpServerResponse from "../http/HttpServerResponse.ts";
import * as Layer from "../Layer.ts";
/**
 * OTLP signal type served by the proxy.
 *
 * @stability unstable
 * @category models
 * @since 4.0.7
 */
export type Signal = "traces" | "logs" | "metrics";
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
export declare const make: (options: {
    readonly baseUrl: string;
    readonly urls?: Partial<Record<Signal, string>> | undefined;
    readonly headers?: Headers.Input | undefined;
    readonly maxBodySize?: ByteSize.Input | undefined;
    readonly timeout?: Duration.Input | undefined;
}) => Effect.Effect<Record<Signal, Effect.Effect<HttpServerResponse.HttpServerResponse, never, HttpServerRequest.HttpServerRequest>>, never, HttpClient.HttpClient>;
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
export declare const layer: (options: {
    readonly baseUrl: string;
    readonly urls?: Partial<Record<Signal, string>> | undefined;
    readonly headers?: Headers.Input | undefined;
    readonly maxBodySize?: ByteSize.Input | undefined;
    readonly timeout?: Duration.Input | undefined;
    readonly prefix?: string | undefined;
    readonly signals?: ReadonlyArray<Signal> | undefined;
}) => Layer.Layer<never, never, HttpRouter.HttpRouter | HttpClient.HttpClient>;
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
export declare const layerFromConfig: (options?: {
    readonly headers?: Headers.Input | undefined;
    readonly maxBodySize?: ByteSize.Input | undefined;
    readonly timeout?: Duration.Input | undefined;
    readonly prefix?: string | undefined;
    readonly signals?: ReadonlyArray<Signal> | undefined;
}) => Layer.Layer<never, never, HttpRouter.HttpRouter | HttpClient.HttpClient>;
//# sourceMappingURL=OtlpProxy.d.ts.map