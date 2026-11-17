/**
 * Configures OpenTelemetry Protocol (OTLP) HTTP export for Effect telemetry.
 *
 * This module installs the OTLP logger, metrics exporter, and tracer exporter
 * from one shared configuration. Use it when an application sends logs, metrics,
 * and traces to the same OpenTelemetry Collector, vendor OTLP endpoint, or local
 * development collector.
 *
 * @stability unstable
 * @since 4.0.0
 */
import type * as Duration from "../Duration.ts";
import type * as Headers from "../http/Headers.ts";
import type * as HttpClient from "../http/HttpClient.ts";
import * as Layer from "../Layer.ts";
import type * as Tracer from "../Tracer.ts";
import type * as OtlpExporter from "./OtlpExporter.ts";
import type { AggregationTemporality } from "./OtlpMetrics.ts";
import * as OtlpSerialization from "./OtlpSerialization.ts";
/**
 * Creates a combined OTLP layer for logs, metrics, and traces.
 *
 * **Details**
 *
 * The layer sends data to `/v1/logs`, `/v1/metrics`, and `/v1/traces` below
 * `baseUrl` and requires an `OtlpSerialization` implementation.
 *
 * The layer provides the `OtlpExporter.Flusher` shared by all three signals,
 * so one `flush` exports their buffered telemetry, for example before a
 * serverless instance is suspended.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layer: (options: {
    readonly baseUrl: string;
    readonly resource?: {
        readonly serviceName?: string | undefined;
        readonly serviceVersion?: string | undefined;
        readonly attributes?: Record<string, unknown>;
    } | undefined;
    readonly headers?: Headers.Input | undefined;
    /**
     * Request compression. Defaults to "none". Gzip uses
     * HttpClientRequest's default minimum body size of 1KB.
     */
    readonly compression?: "gzip" | "none" | undefined;
    readonly maxBatchSize?: number | undefined;
    readonly tracerContext?: (<X>(primitive: Tracer.EffectPrimitive<X>, span: Tracer.AnySpan) => X) | undefined;
    readonly loggerExportInterval?: Duration.Input | undefined;
    readonly loggerExcludeLogSpans?: boolean | undefined;
    readonly loggerMergeWithExisting?: boolean | undefined;
    readonly metricsExportInterval?: Duration.Input | undefined;
    readonly metricsTemporality?: AggregationTemporality | undefined;
    readonly tracerExportInterval?: Duration.Input | undefined;
    readonly shutdownTimeout?: Duration.Input | undefined;
}) => Layer.Layer<OtlpExporter.Flusher, never, HttpClient.HttpClient | OtlpSerialization.OtlpSerialization>;
/**
 * Creates a combined OTLP layer for logs, metrics, and traces from
 * OpenTelemetry configuration.
 *
 * **Details**
 *
 * The layer provides the `OtlpExporter.Flusher` shared by all enabled signals.
 * It is provided even when configuration disables every signal, in which case
 * `flush` is a no-op.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerFromConfig: (options?: {
    readonly resource?: {
        readonly serviceName?: string | undefined;
        readonly serviceVersion?: string | undefined;
        readonly attributes?: Record<string, unknown>;
    } | undefined;
    readonly headers?: Headers.Input | undefined;
    readonly tracerContext?: (<X>(primitive: Tracer.EffectPrimitive<X>, span: Tracer.AnySpan) => X) | undefined;
    readonly loggerExcludeLogSpans?: boolean | undefined;
    readonly loggerMergeWithExisting?: boolean | undefined;
}) => Layer.Layer<OtlpExporter.Flusher, never, HttpClient.HttpClient | OtlpSerialization.OtlpSerialization>;
/**
 * Creates the combined OTLP logs, metrics, and traces layer using JSON
 * serialization.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerJson: (options: {
    readonly baseUrl: string;
    readonly resource?: {
        readonly serviceName?: string | undefined;
        readonly serviceVersion?: string | undefined;
        readonly attributes?: Record<string, unknown>;
    } | undefined;
    readonly headers?: Headers.Input | undefined;
    /**
     * Request compression. Defaults to "none". Gzip uses
     * HttpClientRequest's default minimum body size of 1KB.
     */
    readonly compression?: "gzip" | "none" | undefined;
    readonly maxBatchSize?: number | undefined;
    readonly tracerContext?: (<X>(primitive: Tracer.EffectPrimitive<X>, span: Tracer.AnySpan) => X) | undefined;
    readonly loggerExportInterval?: Duration.Input | undefined;
    readonly loggerExcludeLogSpans?: boolean | undefined;
    readonly loggerMergeWithExisting?: boolean | undefined;
    readonly metricsExportInterval?: Duration.Input | undefined;
    readonly metricsTemporality?: AggregationTemporality | undefined;
    readonly tracerExportInterval?: Duration.Input | undefined;
    readonly shutdownTimeout?: Duration.Input | undefined;
}) => Layer.Layer<OtlpExporter.Flusher, never, HttpClient.HttpClient>;
/**
 * Creates the combined OTLP logs, metrics, and traces layer using protobuf
 * serialization.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerProtobuf: (options: {
    readonly baseUrl: string;
    readonly resource?: {
        readonly serviceName?: string | undefined;
        readonly serviceVersion?: string | undefined;
        readonly attributes?: Record<string, unknown>;
    } | undefined;
    readonly headers?: Headers.Input | undefined;
    /**
     * Request compression. Defaults to "none". Gzip uses
     * HttpClientRequest's default minimum body size of 1KB.
     */
    readonly compression?: "gzip" | "none" | undefined;
    readonly maxBatchSize?: number | undefined;
    readonly tracerContext?: (<X>(primitive: Tracer.EffectPrimitive<X>, span: Tracer.AnySpan) => X) | undefined;
    readonly loggerExportInterval?: Duration.Input | undefined;
    readonly loggerExcludeLogSpans?: boolean | undefined;
    readonly loggerMergeWithExisting?: boolean | undefined;
    readonly metricsExportInterval?: Duration.Input | undefined;
    readonly metricsTemporality?: AggregationTemporality | undefined;
    readonly tracerExportInterval?: Duration.Input | undefined;
    readonly shutdownTimeout?: Duration.Input | undefined;
}) => Layer.Layer<OtlpExporter.Flusher, never, HttpClient.HttpClient>;
//# sourceMappingURL=Otlp.d.ts.map