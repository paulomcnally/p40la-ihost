import * as Redacted from "../../Redacted.js";
/** @internal */
export const unwrap = value => typeof value === "string" ? value : Redacted.value(value);
/** @internal */
export const stripSearchAndHash = url => {
  const queryIndex = url.indexOf("?");
  const hashIndex = url.indexOf("#");
  if (queryIndex === -1) {
    return hashIndex === -1 ? url : url.slice(0, hashIndex);
  }
  if (hashIndex === -1) {
    return url.slice(0, queryIndex);
  }
  return url.slice(0, Math.min(queryIndex, hashIndex));
};
// URL-encoded bodies keep a leading U+FEFF as form data
const bodyDecoder = /*#__PURE__*/new TextDecoder("utf-8", {
  ignoreBOM: true
});
/** @internal */
export const fromBody = bytes =>
// A leading "?" is form data; URLSearchParams would drop it as a query prefix.
new URLSearchParams(bodyDecoder.decode(bytes).replace(/^\?/, "%3F"));
const originForm = /^\/(?![/\\])/;
/** @internal */
export const toPath = url => {
  // Origin-form targets and "*" are already paths; absolute URLs can carry credentials.
  if (url === "*" || originForm.test(url)) {
    return stripSearchAndHash(url);
  }
  try {
    return new URL(url, "http://localhost").pathname;
  } catch {
    return "<invalid URL>";
  }
};
//# sourceMappingURL=urlParams.js.map