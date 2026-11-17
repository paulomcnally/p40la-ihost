import * as Schema_ from "../Schema.ts";
import * as HttpServerResponse from "./HttpServerResponse.ts";
/**
 * The media type of a problem details response body.
 *
 * @stability unstable
 * @category constants
 * @since 4.0.6
 */
export declare const contentType = "application/problem+json";
/**
 * The problem type URI RFC 9457 assigns to problems that only carry an HTTP
 * status.
 *
 * @stability unstable
 * @category constants
 * @since 4.0.6
 */
export declare const blankType = "about:blank";
/**
 * A decoded RFC 9457 problem details object.
 *
 * **Details**
 *
 * The standard members are `type`, `status`, `title`, `detail` and
 * `instance`. Any other member is an extension and is kept as-is.
 *
 * @stability unstable
 * @category models
 * @since 4.0.6
 */
export interface ProblemDetails {
    readonly type: string;
    readonly status?: number;
    readonly title?: string;
    readonly detail?: string;
    readonly instance?: string;
    readonly [extension: string]: unknown;
}
/**
 * The encoded form of a problem details object, where `type` may be omitted.
 *
 * @stability unstable
 * @category models
 * @since 4.0.6
 */
export interface ProblemDetailsEncoded {
    readonly type?: string;
    readonly status?: number;
    readonly title?: string;
    readonly detail?: string;
    readonly instance?: string;
    readonly [extension: string]: unknown;
}
/**
 * The lenient codec for problem details objects.
 *
 * **Details**
 *
 * Decoding follows RFC 9457: a missing `type` defaults to `about:blank`,
 * standard members with the wrong JSON type are ignored, and unknown members
 * are kept as extensions. Encoding always emits `type`.
 *
 * **Example** (Decoding a problem with an extension)
 *
 * ```ts import.meta.vitest
 * import { Schema } from "effect"
 * import { ProblemDetails } from "effect/http"
 *
 * const problem = Schema.decodeUnknownSync(ProblemDetails.Schema)({
 *   status: 404,
 *   detail: "No such user",
 *   traceId: "abc-123"
 * })
 * problem // => { type: "about:blank", status: 404, detail: "No such user", traceId: "abc-123" }
 * ```
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.6
 */
export declare const Schema: Schema_.Codec<ProblemDetails, ProblemDetailsEncoded>;
/**
 * Renders a problem details object as an `application/problem+json` response.
 *
 * **Details**
 *
 * The response status is the problem's `status` member, or `500` when it is
 * absent.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.6
 */
export declare const toResponse: (problem: ProblemDetails) => HttpServerResponse.HttpServerResponse;
//# sourceMappingURL=ProblemDetails.d.ts.map