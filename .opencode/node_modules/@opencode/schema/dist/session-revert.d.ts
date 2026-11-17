import { Schema } from "effect";
export interface Revert extends Schema.Schema.Type<typeof Revert> {
}
export declare const Revert: Schema.Struct<{
    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
    };
    /** Legacy V1 compatibility state. */
    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
        readonly file: Schema.String;
        readonly patch: Schema.String;
        readonly additions: Schema.Int;
        readonly deletions: Schema.Int;
        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
        readonly file: Schema.String;
        readonly patch: Schema.String;
        readonly additions: Schema.Int;
        readonly deletions: Schema.Int;
        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
    }>>>, never, never>;
}>;
export interface RevertV1 extends Schema.Schema.Type<typeof RevertV1> {
}
export declare const RevertV1: Schema.Struct<{
    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
    };
    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
        readonly path: Schema.String;
        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
        readonly additions: Schema.Finite;
        readonly deletions: Schema.Finite;
        readonly patch: Schema.String;
    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
        readonly path: Schema.String;
        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
        readonly additions: Schema.Finite;
        readonly deletions: Schema.Finite;
        readonly patch: Schema.String;
    }>>>, never, never>;
}>;
/** Storage decoder for revert state written before FileDiff became canonical. */
export declare const PersistedRevert: Schema.decodeTo<Schema.toType<Schema.Struct<{
    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
    };
    /** Legacy V1 compatibility state. */
    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
        readonly file: Schema.String;
        readonly patch: Schema.String;
        readonly additions: Schema.Int;
        readonly deletions: Schema.Int;
        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
        readonly file: Schema.String;
        readonly patch: Schema.String;
        readonly additions: Schema.Int;
        readonly deletions: Schema.Int;
        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
    }>>>, never, never>;
}>>, Schema.Union<readonly [Schema.decodeTo<Schema.Struct<{
    readonly source: Schema.tag<"current">;
    readonly revert: Schema.toType<Schema.Struct<{
        readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
            create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
            fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
        };
        /** Legacy V1 compatibility state. */
        readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
        readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
        readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
            readonly file: Schema.String;
            readonly patch: Schema.String;
            readonly additions: Schema.Int;
            readonly deletions: Schema.Int;
            readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
            readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
        }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
            readonly file: Schema.String;
            readonly patch: Schema.String;
            readonly additions: Schema.Int;
            readonly deletions: Schema.Int;
            readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
            readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
        }>>>, never, never>;
    }>>;
}>, Schema.Struct<{
    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
    };
    /** Legacy V1 compatibility state. */
    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
        readonly file: Schema.String;
        readonly patch: Schema.String;
        readonly additions: Schema.Int;
        readonly deletions: Schema.Int;
        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
        readonly file: Schema.String;
        readonly patch: Schema.String;
        readonly additions: Schema.Int;
        readonly deletions: Schema.Int;
        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
    }>>>, never, never>;
}>, never, never>, Schema.decodeTo<Schema.Struct<{
    readonly source: Schema.tag<"legacy">;
    readonly revert: Schema.toType<Schema.Struct<{
        readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
            create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
            fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
        };
        readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
        readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
        readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
        readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
            readonly path: Schema.String;
            readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
            readonly additions: Schema.Finite;
            readonly deletions: Schema.Finite;
            readonly patch: Schema.String;
        }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
            readonly path: Schema.String;
            readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
            readonly additions: Schema.Finite;
            readonly deletions: Schema.Finite;
            readonly patch: Schema.String;
        }>>>, never, never>;
    }>>;
}>, Schema.Struct<{
    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
    };
    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
        readonly path: Schema.String;
        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
        readonly additions: Schema.Finite;
        readonly deletions: Schema.Finite;
        readonly patch: Schema.String;
    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
        readonly path: Schema.String;
        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
        readonly additions: Schema.Finite;
        readonly deletions: Schema.Finite;
        readonly patch: Schema.String;
    }>>>, never, never>;
}>, never, never>]> & {
    readonly tag: "source";
    readonly discriminants: readonly ["current", "legacy"];
    readonly cases: {
        current: Schema.decodeTo<Schema.Struct<{
            readonly source: Schema.tag<"current">;
            readonly revert: Schema.toType<Schema.Struct<{
                readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                    create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                };
                /** Legacy V1 compatibility state. */
                readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
                readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                    readonly file: Schema.String;
                    readonly patch: Schema.String;
                    readonly additions: Schema.Int;
                    readonly deletions: Schema.Int;
                    readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                    readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                    readonly file: Schema.String;
                    readonly patch: Schema.String;
                    readonly additions: Schema.Int;
                    readonly deletions: Schema.Int;
                    readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                    readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                }>>>, never, never>;
            }>>;
        }>, Schema.Struct<{
            readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
            };
            /** Legacy V1 compatibility state. */
            readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
            readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
            readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                readonly file: Schema.String;
                readonly patch: Schema.String;
                readonly additions: Schema.Int;
                readonly deletions: Schema.Int;
                readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
            }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                readonly file: Schema.String;
                readonly patch: Schema.String;
                readonly additions: Schema.Int;
                readonly deletions: Schema.Int;
                readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
            }>>>, never, never>;
        }>, never, never>;
        legacy: Schema.decodeTo<Schema.Struct<{
            readonly source: Schema.tag<"legacy">;
            readonly revert: Schema.toType<Schema.Struct<{
                readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                    create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                };
                readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                    readonly path: Schema.String;
                    readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                    readonly additions: Schema.Finite;
                    readonly deletions: Schema.Finite;
                    readonly patch: Schema.String;
                }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                    readonly path: Schema.String;
                    readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                    readonly additions: Schema.Finite;
                    readonly deletions: Schema.Finite;
                    readonly patch: Schema.String;
                }>>>, never, never>;
            }>>;
        }>, Schema.Struct<{
            readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
            };
            readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
            readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
            readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
            readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                readonly path: Schema.String;
                readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                readonly additions: Schema.Finite;
                readonly deletions: Schema.Finite;
                readonly patch: Schema.String;
            }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                readonly path: Schema.String;
                readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                readonly additions: Schema.Finite;
                readonly deletions: Schema.Finite;
                readonly patch: Schema.String;
            }>>>, never, never>;
        }>, never, never>;
    };
    readonly isAnyOf: <const Keys>(keys: readonly Keys[]) => (value: {
        readonly source: "current";
        readonly revert: {
            readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
            readonly partID?: string | undefined;
            readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
            readonly files?: readonly {
                readonly file: string;
                readonly patch: string;
                readonly additions: number;
                readonly deletions: number;
                readonly status: "added" | "deleted" | "modified";
                readonly more?: true | undefined;
            }[] | undefined;
        };
    } | {
        readonly source: "legacy";
        readonly revert: {
            readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
            readonly partID?: string | undefined;
            readonly snapshot?: string | undefined;
            readonly diff?: string | undefined;
            readonly files?: readonly {
                readonly path: string;
                readonly status: "added" | "deleted" | "modified";
                readonly additions: number;
                readonly deletions: number;
                readonly patch: string;
            }[] | undefined;
        };
    }) => value is Extract<{
        readonly source: "current";
        readonly revert: {
            readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
            readonly partID?: string | undefined;
            readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
            readonly files?: readonly {
                readonly file: string;
                readonly patch: string;
                readonly additions: number;
                readonly deletions: number;
                readonly status: "added" | "deleted" | "modified";
                readonly more?: true | undefined;
            }[] | undefined;
        };
    }, {
        readonly source: Keys;
    }> | Extract<{
        readonly source: "legacy";
        readonly revert: {
            readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
            readonly partID?: string | undefined;
            readonly snapshot?: string | undefined;
            readonly diff?: string | undefined;
            readonly files?: readonly {
                readonly path: string;
                readonly status: "added" | "deleted" | "modified";
                readonly additions: number;
                readonly deletions: number;
                readonly patch: string;
            }[] | undefined;
        };
    }, {
        readonly source: Keys;
    }>;
    readonly guards: {
        current: (u: unknown) => u is {
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        };
        legacy: (u: unknown) => u is {
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        };
    };
    readonly match: {
        <Handlers extends { [K in "current" | "legacy"]: (value: {
            readonly current: Schema.decodeTo<Schema.Struct<{
                readonly source: Schema.tag<"current">;
                readonly revert: Schema.toType<Schema.Struct<{
                    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    };
                    /** Legacy V1 compatibility state. */
                    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
                    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                        readonly file: Schema.String;
                        readonly patch: Schema.String;
                        readonly additions: Schema.Int;
                        readonly deletions: Schema.Int;
                        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                        readonly file: Schema.String;
                        readonly patch: Schema.String;
                        readonly additions: Schema.Int;
                        readonly deletions: Schema.Int;
                        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                    }>>>, never, never>;
                }>>;
            }>, Schema.Struct<{
                readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                    create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                };
                /** Legacy V1 compatibility state. */
                readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
                readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                    readonly file: Schema.String;
                    readonly patch: Schema.String;
                    readonly additions: Schema.Int;
                    readonly deletions: Schema.Int;
                    readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                    readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                    readonly file: Schema.String;
                    readonly patch: Schema.String;
                    readonly additions: Schema.Int;
                    readonly deletions: Schema.Int;
                    readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                    readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                }>>>, never, never>;
            }>, never, never>;
            readonly legacy: Schema.decodeTo<Schema.Struct<{
                readonly source: Schema.tag<"legacy">;
                readonly revert: Schema.toType<Schema.Struct<{
                    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    };
                    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                        readonly path: Schema.String;
                        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                        readonly additions: Schema.Finite;
                        readonly deletions: Schema.Finite;
                        readonly patch: Schema.String;
                    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                        readonly path: Schema.String;
                        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                        readonly additions: Schema.Finite;
                        readonly deletions: Schema.Finite;
                        readonly patch: Schema.String;
                    }>>>, never, never>;
                }>>;
            }>, Schema.Struct<{
                readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                    create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                };
                readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                    readonly path: Schema.String;
                    readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                    readonly additions: Schema.Finite;
                    readonly deletions: Schema.Finite;
                    readonly patch: Schema.String;
                }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                    readonly path: Schema.String;
                    readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                    readonly additions: Schema.Finite;
                    readonly deletions: Schema.Finite;
                    readonly patch: Schema.String;
                }>>>, never, never>;
            }>, never, never>;
        }[K]["Type"]) => any; }>(value: {
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        } | {
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        }, cases: Handlers): import("effect/Unify").Unify<ReturnType<Handlers["current" | "legacy"]>>;
        <Handlers extends { [K in "current" | "legacy"]: (value: {
            readonly current: Schema.decodeTo<Schema.Struct<{
                readonly source: Schema.tag<"current">;
                readonly revert: Schema.toType<Schema.Struct<{
                    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    };
                    /** Legacy V1 compatibility state. */
                    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
                    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                        readonly file: Schema.String;
                        readonly patch: Schema.String;
                        readonly additions: Schema.Int;
                        readonly deletions: Schema.Int;
                        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                        readonly file: Schema.String;
                        readonly patch: Schema.String;
                        readonly additions: Schema.Int;
                        readonly deletions: Schema.Int;
                        readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                        readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                    }>>>, never, never>;
                }>>;
            }>, Schema.Struct<{
                readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                    create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                };
                /** Legacy V1 compatibility state. */
                readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.brand<Schema.String, "Snapshot.ID">>>, Schema.optionalKey<Schema.brand<Schema.String, "Snapshot.ID">>, never, never>;
                readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                    readonly file: Schema.String;
                    readonly patch: Schema.String;
                    readonly additions: Schema.Int;
                    readonly deletions: Schema.Int;
                    readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                    readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                    readonly file: Schema.String;
                    readonly patch: Schema.String;
                    readonly additions: Schema.Int;
                    readonly deletions: Schema.Int;
                    readonly status: Schema.Literals<readonly ["added", "deleted", "modified"]>;
                    readonly more: Schema.decodeTo<Schema.optional<Schema.toType<Schema.Literal<true>>>, Schema.optionalKey<Schema.Literal<true>>, never, never>;
                }>>>, never, never>;
            }>, never, never>;
            readonly legacy: Schema.decodeTo<Schema.Struct<{
                readonly source: Schema.tag<"legacy">;
                readonly revert: Schema.toType<Schema.Struct<{
                    readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                        create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                        fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    };
                    readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                    readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                        readonly path: Schema.String;
                        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                        readonly additions: Schema.Finite;
                        readonly deletions: Schema.Finite;
                        readonly patch: Schema.String;
                    }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                        readonly path: Schema.String;
                        readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                        readonly additions: Schema.Finite;
                        readonly deletions: Schema.Finite;
                        readonly patch: Schema.String;
                    }>>>, never, never>;
                }>>;
            }>, Schema.Struct<{
                readonly messageID: Schema.brand<Schema.String, "Session.Message.ID"> & {
                    create: () => string & import("effect/Brand").Brand<"Session.Message.ID">;
                    fromEvent: (eventID: string) => string & import("effect/Brand").Brand<"Session.Message.ID">;
                };
                readonly partID: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly snapshot: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly diff: Schema.decodeTo<Schema.optional<Schema.toType<Schema.String>>, Schema.optionalKey<Schema.String>, never, never>;
                readonly files: Schema.decodeTo<Schema.optional<Schema.toType<Schema.$Array<Schema.Struct<{
                    readonly path: Schema.String;
                    readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                    readonly additions: Schema.Finite;
                    readonly deletions: Schema.Finite;
                    readonly patch: Schema.String;
                }>>>>, Schema.optionalKey<Schema.$Array<Schema.Struct<{
                    readonly path: Schema.String;
                    readonly status: Schema.Literals<readonly ["added", "modified", "deleted"]>;
                    readonly additions: Schema.Finite;
                    readonly deletions: Schema.Finite;
                    readonly patch: Schema.String;
                }>>>, never, never>;
            }>, never, never>;
        }[K]["Type"]) => any; }>(cases: Handlers): (value: {
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        } | {
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        }) => import("effect/Unify").Unify<ReturnType<Handlers["current" | "legacy"]>>;
    };
    readonly matchOrElse: {
        <Handlers, OrElse>(value: {
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        } | {
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        }, cases: Handlers & {
            readonly current?: ((value: {
                readonly source: "current";
                readonly revert: {
                    readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                    readonly partID?: string | undefined;
                    readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                    readonly files?: readonly {
                        readonly file: string;
                        readonly patch: string;
                        readonly additions: number;
                        readonly deletions: number;
                        readonly status: "added" | "deleted" | "modified";
                        readonly more?: true | undefined;
                    }[] | undefined;
                };
            }) => any) | undefined;
            readonly legacy?: ((value: {
                readonly source: "legacy";
                readonly revert: {
                    readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                    readonly partID?: string | undefined;
                    readonly snapshot?: string | undefined;
                    readonly diff?: string | undefined;
                    readonly files?: readonly {
                        readonly path: string;
                        readonly status: "added" | "deleted" | "modified";
                        readonly additions: number;
                        readonly deletions: number;
                        readonly patch: string;
                    }[] | undefined;
                };
            }) => any) | undefined;
        } & (Exclude<keyof Handlers, "current" | "legacy"> extends never ? unknown : never), orElse: OrElse & ((value: Exclude<{
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        }, {
            readonly source: keyof Handlers;
        }> | Exclude<{
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        }, {
            readonly source: keyof Handlers;
        }>) => any)): import("effect/Unify").Unify<{ [K in keyof Handlers]-?: NonNullable<Handlers[K]> extends (...args: Array<any>) => infer R ? R : never; }[keyof Handlers] | (OrElse extends (...args: Array<any>) => infer R ? R : never)>;
        <Handlers, OrElse>(cases: Handlers & {
            readonly current?: ((value: {
                readonly source: "current";
                readonly revert: {
                    readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                    readonly partID?: string | undefined;
                    readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                    readonly files?: readonly {
                        readonly file: string;
                        readonly patch: string;
                        readonly additions: number;
                        readonly deletions: number;
                        readonly status: "added" | "deleted" | "modified";
                        readonly more?: true | undefined;
                    }[] | undefined;
                };
            }) => any) | undefined;
            readonly legacy?: ((value: {
                readonly source: "legacy";
                readonly revert: {
                    readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                    readonly partID?: string | undefined;
                    readonly snapshot?: string | undefined;
                    readonly diff?: string | undefined;
                    readonly files?: readonly {
                        readonly path: string;
                        readonly status: "added" | "deleted" | "modified";
                        readonly additions: number;
                        readonly deletions: number;
                        readonly patch: string;
                    }[] | undefined;
                };
            }) => any) | undefined;
        } & (Exclude<keyof Handlers, "current" | "legacy"> extends never ? unknown : never), orElse: OrElse & ((value: Exclude<{
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        }, {
            readonly source: keyof Handlers;
        }> | Exclude<{
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        }, {
            readonly source: keyof Handlers;
        }>) => any)): (value: {
            readonly source: "current";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: (string & import("effect/Brand").Brand<"Snapshot.ID">) | undefined;
                readonly files?: readonly {
                    readonly file: string;
                    readonly patch: string;
                    readonly additions: number;
                    readonly deletions: number;
                    readonly status: "added" | "deleted" | "modified";
                    readonly more?: true | undefined;
                }[] | undefined;
            };
        } | {
            readonly source: "legacy";
            readonly revert: {
                readonly messageID: string & import("effect/Brand").Brand<"Session.Message.ID">;
                readonly partID?: string | undefined;
                readonly snapshot?: string | undefined;
                readonly diff?: string | undefined;
                readonly files?: readonly {
                    readonly path: string;
                    readonly status: "added" | "deleted" | "modified";
                    readonly additions: number;
                    readonly deletions: number;
                    readonly patch: string;
                }[] | undefined;
            };
        }) => import("effect/Unify").Unify<{ [K in keyof Handlers]-?: NonNullable<Handlers[K]> extends (...args: Array<any>) => infer R ? R : never; }[keyof Handlers] | (OrElse extends (...args: Array<any>) => infer R ? R : never)>;
    };
}, never, never>;
