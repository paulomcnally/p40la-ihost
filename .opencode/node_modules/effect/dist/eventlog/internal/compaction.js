import * as Effect from "../../Effect.js";
/** Rules and audit logs shared by the memory and SQL compaction storages. */
export const hourMillis = 60 * 60 * 1000;
/** The newest storage-time hour that ended at least `lagMillis` ago. */
export const lagHour = (now, lagMillis) => Math.floor((now - lagMillis) / hourMillis) - 1;
/** A staged batch's position packs its segment index and its index in the segment. */
export const positionsPerSegment = 2 ** 20;
export const maxSegments = /*#__PURE__*/Math.floor(Number.MAX_SAFE_INTEGER / positionsPerSegment);
const service = "EventLogServerEncrypted";
export const logCommit = options => Effect.logInfo(options.result === "Committed" ? "Committed compaction snapshot" : "Compaction snapshot superseded").pipe(Effect.annotateLogs({
  service,
  publicKey: options.publicKey,
  storeId: options.storeId,
  keptCount: options.keptCount,
  through: options.through,
  activationId: options.activationId
}));
export const logNoOp = options => Effect.logInfo("Committed compaction no-op").pipe(Effect.annotateLogs({
  service,
  ...options
}));
export const logRollback = options => Effect.logInfo("Rolled back compaction snapshot").pipe(Effect.annotateLogs({
  service,
  publicKey: options.publicKey,
  storeId: options.storeId,
  keptCount: options.activation.keptCount,
  base: options.activation.base,
  activationId: options.activation.id,
  withdrawnActivationId: options.withdrawn.id,
  withdrawnBase: options.withdrawn.base,
  withdrawnKeptCount: options.withdrawn.keptCount
}));
//# sourceMappingURL=compaction.js.map