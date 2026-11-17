/**
 * Redis support shared by persistence modules.
 *
 * This module defines a `Redis` service that can send Redis commands, subscribe
 * to pub/sub channels, run Lua scripts, and reserve a dedicated connection from
 * a pool. It does not create a Redis client itself; callers provide the
 * client-specific operations. The module also provides helpers for describing
 * Lua scripts, loading them once, and running them later by their cached Redis
 * id.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Cache from "../Cache.js";
import * as Cause from "../Cause.js";
import * as Context from "../Context.js";
import * as Duration from "../Duration.js";
import * as Effect from "../Effect.js";
import * as Equal from "../Equal.js";
import * as Exit from "../Exit.js";
import { constant, identity } from "../Function.js";
import * as Hash from "../Hash.js";
import * as Pool from "../Pool.js";
import * as Queue from "../Queue.js";
import * as Schema from "../Schema.js";
import * as Scope from "../Scope.js";
/**
 * Service for sending Redis commands, subscribing to channels, and evaluating
 * cached Lua scripts.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export class Redis extends /*#__PURE__*/Context.Service()("effect/persistence/Redis") {}
/**
 * Creates a `Redis` service from raw command and subscription operations.
 *
 * **Details**
 *
 * Lua scripts are loaded through `SCRIPT LOAD`, cached, and then invoked with
 * `EVALSHA`.
 *
 * `connect` opens the dedicated connections behind `reserve`. Its `close` must
 * fail any command still in flight on that connection. Connections are pooled
 * according to `pool`, and closing the scope closes every connection,
 * including reserved ones with a command in flight.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const make = /*#__PURE__*/Effect.fnUntraced(function* (options) {
  const scriptCache = yield* Cache.makeWith(script => options.send("SCRIPT", "LOAD", script.lua), {
    capacity: Number.POSITIVE_INFINITY,
    timeToLive: exit => Exit.isSuccess(exit) ? Duration.infinity : Duration.zero
  });
  const makeEval = send => script => (...params) => {
    const evalSha = sha => send("EVALSHA", sha, script.numberOfKeys(...params).toString(), ...script.params(...params).map(param => String(param)));
    return Cache.get(scriptCache, script).pipe(Effect.flatMap(evalSha), Effect.catchIf(error => String(error.cause).includes("NOSCRIPT"), () => Cache.refresh(scriptCache, script).pipe(Effect.flatMap(evalSha))));
  };
  // Closing a connection fails its in-flight command, which releases the
  // reservation. The pool waits for reservations on shutdown, so connections
  // are closed before it shuts down.
  const connections = new Set();
  const pool = yield* Pool.makeWithTTL({
    acquire: Effect.acquireRelease(Effect.map(options.connect, connection => {
      const close = Effect.suspend(() => connections.delete(close) ? connection.close : Effect.void);
      connections.add(close);
      return {
        send: connection.send,
        close,
        healthy: true,
        active: 0
      };
    }), connection => connection.close),
    min: options.pool?.minConnections ?? 0,
    max: options.pool?.maxConnections ?? 10,
    concurrency: 1,
    timeToLive: options.pool?.idleTimeout ?? Duration.seconds(60)
  });
  yield* Effect.addFinalizer(() => Effect.forEach(connections, identity, {
    discard: true
  }));
  const acquireTimeout = options.pool?.acquireTimeout;
  const get = acquireTimeout === undefined ? Pool.get(pool) : Pool.get(pool).pipe(Effect.timeoutOrElse({
    duration: acquireTimeout,
    orElse: () => Effect.fail(new RedisError({
      cause: new Cause.TimeoutError("Timed out waiting for a reserved Redis connection")
    }))
  }));
  const reserve = Effect.gen(function* () {
    const connection = yield* get;
    yield* Effect.addFinalizer(() => connection.healthy && connection.active === 0 ? Effect.void : Pool.invalidate(pool, connection));
    const send = (command, ...args) => Effect.suspend(() => {
      connection.active++;
      return connection.send(command, ...args).pipe(Effect.onExit(exit => {
        connection.active--;
        if (Exit.isFailure(exit)) connection.healthy = false;
        return Effect.void;
      }));
    });
    return identity({
      send,
      eval: makeEval(send)
    });
  });
  const subscribe = Effect.fnUntraced(function* (channel) {
    const queue = yield* Queue.unbounded();
    yield* Scope.addFinalizer(yield* Effect.scope, Queue.shutdown(queue));
    const onFailure = cause => Queue.failCause(queue, cause).pipe(Effect.asVoid);
    yield* options.subscribe(channel, message => {
      Queue.offerUnsafe(queue, message);
    }).pipe(Effect.flatMap(listen => Effect.forkScoped(Effect.catchCause(listen, onFailure))), Effect.catchCause(onFailure));
    return queue;
  });
  return identity({
    send: options.send,
    subscribe,
    eval: makeEval(options.send),
    reserve
  });
});
const ErrorTypeId = "~effect/persistence/Redis/RedisError";
/**
 * Error raised by Redis command or script execution.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class RedisError extends /*#__PURE__*/Schema.Error(ErrorTypeId)({
  _tag: /*#__PURE__*/Schema.tag("RedisError"),
  cause: /*#__PURE__*/Schema.Defect()
}) {
  /**
   * Marks this value as a Redis persistence error for runtime guards.
   *
   * @stability unstable
   * @since 4.0.0
   */
  [ErrorTypeId] = ErrorTypeId;
}
const ScriptTypeId = "~effect/persistence/Redis/Script";
const ScriptProto = {
  [ScriptTypeId]: {
    params: identity,
    result: identity
  },
  withReturnType() {
    return this;
  },
  [Equal.symbol](that) {
    return this === that;
  },
  [Hash.symbol]() {
    return Hash.random(this);
  }
};
/**
 * Constructs a typed Redis Lua script descriptor.
 *
 * **Details**
 *
 * The result type defaults to `void` and can be refined with
 * `withReturnType`.
 *
 * @stability unstable
 * @category scripting
 * @since 4.0.0
 */
export const script = (f, options) => Object.setPrototypeOf({
  ...options,
  params: f,
  numberOfKeys: typeof options.numberOfKeys === "number" ? constant(options.numberOfKeys) : options.numberOfKeys
}, ScriptProto);
//# sourceMappingURL=Redis.js.map