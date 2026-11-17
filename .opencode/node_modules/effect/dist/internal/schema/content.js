const ContentTypeId = /*#__PURE__*/Symbol.for("~effect/Schema/content");
/** @internal */
export function setContent(operation, content) {
  Object.defineProperty(operation, ContentTypeId, {
    value: content
  });
  return operation;
}
/** @internal */
export function getContent(operation) {
  return operation[ContentTypeId];
}
//# sourceMappingURL=content.js.map