/** @internal */
export const getAllObjectKeys = obj => {
  const keys = new Set(Reflect.ownKeys(obj));
  if (obj.constructor === Object) return keys;
  if (obj instanceof Error) {
    keys.delete("stack");
  }
  const proto = Object.getPrototypeOf(obj);
  let current = proto;
  while (current !== null && current !== Object.prototype) {
    const ownKeys = Reflect.ownKeys(current);
    for (let i = 0; i < ownKeys.length; i++) {
      keys.add(ownKeys[i]);
    }
    current = Object.getPrototypeOf(current);
  }
  if (keys.has("constructor") && typeof obj.constructor === "function" && proto === obj.constructor.prototype) {
    keys.delete("constructor");
  }
  return keys;
};
/** @internal */
export const byReferenceInstances = /*#__PURE__*/new WeakSet();
/** @internal */
export const viewBytes = view => new Uint8Array(view.buffer, view.byteOffset, view.byteLength);
/** @internal */
export const isArrayBufferLike = u => u instanceof ArrayBuffer || typeof SharedArrayBuffer !== "undefined" && u instanceof SharedArrayBuffer;
/**
 * Returns `undefined` for detached buffers, which can no longer be read.
 *
 * @internal
 */
export const bufferBytes = buffer => {
  try {
    return new Uint8Array(buffer);
  } catch {
    return undefined;
  }
};
//# sourceMappingURL=equal.js.map