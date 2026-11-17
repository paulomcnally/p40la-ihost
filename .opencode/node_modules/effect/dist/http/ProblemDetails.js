/**
 * RFC 9457 Problem Details values for HTTP responses.
 *
 * A problem details object is the `application/problem+json` body a server
 * sends to describe an error. This module provides the reusable value shape,
 * a lenient codec that follows the parsing rules of RFC 9457, and a response
 * constructor. The `effect/http-api` modules build typed problem declarations,
 * built-in error rendering and OpenAPI output on top of it.
 *
 * @stability unstable
 * @since 4.0.6
 */
import * as Effect from "../Effect.js";
import * as Schema_ from "../Schema.js";
import * as HttpServerResponse from "./HttpServerResponse.js";
/**
 * The media type of a problem details response body.
 *
 * @stability unstable
 * @category constants
 * @since 4.0.6
 */
export const contentType = "application/problem+json";
/**
 * The problem type URI RFC 9457 assigns to problems that only carry an HTTP
 * status.
 *
 * @stability unstable
 * @category constants
 * @since 4.0.6
 */
export const blankType = "about:blank";
/**
 * A standard member that is dropped when it is missing or has the wrong JSON
 * type, as RFC 9457 section 3.1 requires.
 *
 * @internal
 */
export const optionalMember = schema => Schema_.optionalKey(schema).pipe(Schema_.catchDecoding(() => Effect.succeedNone));
const withFallback = (schema, value) => Schema_.withDecodingDefaultKey(Effect.succeed(value))(schema).pipe(Schema_.catchDecoding(() => Effect.succeedSome(value)));
/**
 * A required member that decodes to a fixed value when it is missing or has
 * the wrong JSON type. The value is also the constructor default.
 *
 * @internal
 */
export const fixedMember = value => withFallback(Schema_.tag(value), value);
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
export const Schema = /*#__PURE__*/Schema_.StructWithRest(/*#__PURE__*/Schema_.Struct({
  type: /*#__PURE__*/withFallback(Schema_.String, blankType),
  status: /*#__PURE__*/optionalMember(Schema_.Number),
  title: /*#__PURE__*/optionalMember(Schema_.String),
  detail: /*#__PURE__*/optionalMember(Schema_.String),
  instance: /*#__PURE__*/optionalMember(Schema_.String)
}), [/*#__PURE__*/Schema_.Record(Schema_.String, Schema_.Unknown)]);
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
export const toResponse = problem => HttpServerResponse.jsonUnsafe(problem, {
  status: problem.status ?? 500,
  contentType
});
//# sourceMappingURL=ProblemDetails.js.map