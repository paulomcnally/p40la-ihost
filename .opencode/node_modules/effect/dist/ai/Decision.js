/**
 * Defines decisions that a `DecisionModel` answers over a single input.
 * Pair an input schema with named classification, rating, probability, or
 * locate decisions using `make`, then answer them with `DecisionModel.decide`,
 * or with `DecisionModel.decideSettled` to keep valid answers when others fail.
 *
 * @see {@link make} for building a definition from an input schema and decisions
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Base64 from "../encoding/Base64.js";
import * as Predicate from "../Predicate.js";
import * as Schema from "../Schema.js";
import * as SchemaTransformation from "../SchemaTransformation.js";
/**
 * Brand for decision definitions.
 *
 * @stability unstable
 * @category type IDs
 * @since 4.0.0
 */
export const TypeId = "~effect/ai/Decision";
/**
 * Key that marks an encoded {@link Image}.
 *
 * @stability unstable
 * @category models
 * @since 4.1.0
 */
export const ImageKey = "~effect/ai/Decision/Image";
const isImage = u => Predicate.isObject(u) && Predicate.isString(u.mediaType) && (Predicate.isString(u.data) || u.data instanceof globalThis.Uint8Array || u.data instanceof globalThis.URL);
/**
 * Schema for an image input field. Its type is `DecisionModel.Image`: a
 * media type with base64 data, bytes, or a URL. Decisions refer to the image
 * by the field's path, like any other field.
 *
 * **Details**
 *
 * The field encodes to an {@link ImageEncoded} object. `DecisionModel.decide`
 * replaces each one in the encoded input with a reference string and sends the
 * images to the provider separately; an image used in several fields is sent
 * once. The encoded shape also appears in JSON Schema generated from the input.
 *
 * **Example** (Declaring an image field)
 *
 * ```ts
 * import { Schema } from "effect"
 * import { Decision } from "effect/ai"
 *
 * const Listing = Schema.Struct({
 *   title: Schema.String,
 *   photo: Decision.Image
 * })
 * ```
 *
 * @stability unstable
 * @category schemas
 * @since 4.1.0
 */
export const Image = /*#__PURE__*/Schema.Struct({
  [ImageKey]: Schema.Union([Schema.Struct({
    mediaType: Schema.String,
    base64: Schema.String
  }), Schema.Struct({
    mediaType: Schema.String,
    url: Schema.URLFromString
  })])
}).pipe(/*#__PURE__*/Schema.decodeTo(/*#__PURE__*/Schema.declare(isImage, {
  expected: "DecisionModel.Image"
}), /*#__PURE__*/SchemaTransformation.transform({
  decode: ({
    [ImageKey]: image
  }) => ({
    mediaType: image.mediaType,
    data: "base64" in image ? image.base64 : image.url
  }),
  encode: ({
    mediaType,
    data
  }) => ({
    [ImageKey]: data instanceof globalThis.URL ? {
      mediaType,
      url: data
    } : {
      mediaType,
      base64: typeof data === "string" ? data : Base64.encode(data)
    }
  })
})));
/**
 * Creates a classification decision from labelled criteria.
 * Descriptions can be `null` when the label needs no explanation.
 * Throws if fewer than two labels are supplied.
 *
 * **Example** (Choosing a department)
 *
 * ```ts
 * import { Decision } from "effect/ai"
 *
 * const department = Decision.classify({
 *   instructions: "Which team should handle this",
 *   criteria: {
 *     billing: "payments",
 *     technical: "bugs",
 *     sales: "pricing"
 *   }
 * })
 * ```
 *
 * @see {@link rate} for an ordered scale
 * @see {@link probability} for a single yes or no likelihood
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const classify = options => {
  if (Object.keys(options.criteria).length < 2) {
    throw new Error("Decision.classify: criteria must contain at least two labels");
  }
  return {
    _tag: "Classify",
    instructions: options.instructions,
    criteria: options.criteria
  };
};
/**
 * Creates a rating decision from an ordered list of criteria.
 * Strings serve as both label and description; use `{ label, description }`
 * to separate them. Answers use the label.
 * Throws if fewer than two levels or duplicate labels are supplied.
 *
 * **Example** (Rating frustration)
 *
 * ```ts
 * import { Decision } from "effect/ai"
 *
 * const frustration = Decision.rate({
 *   instructions: "How frustrated",
 *   criteria: ["calm", "frustrated", "angry"]
 * })
 * ```
 *
 * **Example** (Describing levels with structure)
 *
 * ```ts
 * import { Decision } from "effect/ai"
 *
 * const severity = Decision.rate({
 *   instructions: "How severe is the reported issue?",
 *   criteria: [
 *     "cosmetic",
 *     {
 *       label: "degraded",
 *       description: {
 *         what: "Broken or degraded feature, but workaround exists",
 *         examples: ["export fails in one browser but works in another"]
 *       }
 *     },
 *     {
 *       label: "blocking",
 *       description: {
 *         what: "Blocking issue; no workaround exists",
 *         examples: ["cannot log in", "data loss"]
 *       }
 *     }
 *   ]
 * })
 * ```
 *
 * @see {@link classify} for unordered labels
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const rate = options => {
  if (options.criteria.length < 2) {
    throw new Error("Decision.rate: criteria must contain at least two levels");
  }
  const criteria = options.criteria.map(level => typeof level === "string" ? {
    label: level,
    description: level
  } : level);
  if (new Set(criteria.map(level => level.label)).size !== criteria.length) {
    throw new Error("Decision.rate: criteria must contain distinct levels");
  }
  return {
    _tag: "Rate",
    instructions: options.instructions,
    criteria
  };
};
/**
 * Creates a probability decision from instructions and optional outcome descriptions.
 * When `criteria` is supplied, descriptions for both `false` and `true` are required.
 *
 * **Example** (Estimating urgency)
 *
 * ```ts
 * import { Decision } from "effect/ai"
 *
 * const urgent = Decision.probability({
 *   instructions: "The message is time-sensitive"
 * })
 * ```
 *
 * **Example** (Providing outcome descriptions)
 *
 * ```ts
 * import { Decision } from "effect/ai"
 *
 * const urgent = Decision.probability({
 *   instructions: "The message is time-sensitive",
 *   criteria: {
 *     false: "The message can wait",
 *     true: "The message needs immediate attention"
 *   }
 * })
 * ```
 *
 * @see {@link classify} for more than two outcomes
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const probability = options => ({
  _tag: "Probability",
  instructions: options.instructions,
  criteria: options.criteria
});
/**
 * Creates a decision that finds one target in an image field of the input.
 * `image` is the field's path in the encoded input: keys joined with `.` and
 * array indices in brackets, such as `"frames[2].frame"`. The model must be
 * created with `supportsLocate: true`.
 *
 * **Details**
 *
 * Paths use encoded key names, so a key renamed with `Schema.encodeKeys` is
 * targeted by its encoded name. `make` accepts only paths to required
 * {@link Image} fields, and `DecisionModel.decide` fails with
 * `AiError.InvalidUserInputError` when the path holds no image, such as an
 * index past the end of an array.
 *
 * The answer is always one box, even when the target is absent. Pair the
 * decision with a {@link probability} decision that the target is present.
 *
 * **Gotchas**
 *
 * - For a generic input schema, the compiler error for an invalid path names
 *   the whole decision type rather than the path.
 *
 * **Example** (Finding a mug in a photo)
 *
 * ```ts
 * import { Schema } from "effect"
 * import { Decision } from "effect/ai"
 *
 * const FindMug = Decision.make({
 *   input: Schema.Struct({ photo: Decision.Image }),
 *   decisions: {
 *     present: Decision.probability({ instructions: "`photo` shows a coffee mug" }),
 *     mug: Decision.locate({ image: "photo", instructions: "the coffee mug" })
 *   }
 * })
 * ```
 *
 * @see {@link LocateAnswer} for the answer
 *
 * @stability unstable
 * @category constructors
 * @since 4.1.0
 */
export const locate = options => ({
  _tag: "Locate",
  image: options.image,
  instructions: options.instructions
});
/**
 * Creates a decision definition from an input schema and named decisions.
 * `DecisionModel.decide` encodes the input with `Schema.toCodecJson` before
 * calling the provider. Answer keys and types are inferred from the decisions.
 * Throws if `decisions` is empty.
 *
 * **Example** (Defining ticket triage)
 *
 * ```ts
 * import { Schema } from "effect"
 * import { Decision } from "effect/ai"
 *
 * const Ticket = Schema.Struct({
 *   subject: Schema.String,
 *   body: Schema.String
 * })
 *
 * const TicketTriage = Decision.make({
 *   input: Ticket,
 *   decisions: {
 *     department: Decision.classify({
 *       instructions: "Which team should handle this",
 *       criteria: { billing: "payments", technical: "bugs" }
 *     }),
 *     urgent: Decision.probability({
 *       instructions: "The message is time-sensitive",
 *       criteria: { false: "No time pressure", true: "Needs action now" }
 *     })
 *   }
 * })
 * ```
 *
 * @see {@link Definition} for the returned shape
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const make = options => {
  if (Object.keys(options.decisions).length === 0) {
    throw new Error("Decision.make: decisions must not be empty");
  }
  return {
    [TypeId]: TypeId,
    input: options.input,
    decisions: options.decisions
  };
};
//# sourceMappingURL=Decision.js.map