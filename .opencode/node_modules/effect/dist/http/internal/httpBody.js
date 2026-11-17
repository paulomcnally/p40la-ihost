import * as Headers from "../Headers.js";
/**
 * @internal
 */
export const updateHeaders = (headers, body) => {
  if (body._tag === "Empty" || body._tag === "FormData") {
    return Headers.remove(Headers.remove(headers, "content-type"), "content-length");
  }
  headers = body.contentType === undefined ? Headers.remove(headers, "content-type") : Headers.set(headers, "content-type", body.contentType);
  return updateContentLength(headers, body);
};
/**
 * @internal
 */
export const updateContentLength = (headers, body) => {
  const contentLength = body._tag === "Empty" || body._tag === "FormData" ? undefined : body.contentLength;
  return contentLength === undefined ? Headers.remove(headers, "content-length") : Headers.set(headers, "content-length", contentLength.toString());
};
/**
 * @internal
 */
export const parseContentLength = contentLength => {
  if (contentLength === null || contentLength === undefined || !/^\d+$/.test(contentLength)) {
    return undefined;
  }
  const parsed = Number(contentLength);
  return Number.isSafeInteger(parsed) ? parsed : undefined;
};
//# sourceMappingURL=httpBody.js.map