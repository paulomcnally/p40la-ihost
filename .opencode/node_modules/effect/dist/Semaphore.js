import { dual, identity } from "./Function.js";
import * as core from "./internal/core.js";
import * as internal from "./internal/effect.js";
/**
 * Creates a `Semaphore` synchronously with the specified total
 * number of permits.
 *
 * **When to use**
 *
 * Use to construct a semaphore synchronously when an immediate value is
 * required outside an Effect workflow.
 *
 * **Example** (Creating an unsafe semaphore)
 *
 * ```ts import.meta.vitest
 * import { Effect, Semaphore } from "effect"
 *
 * const semaphore = Semaphore.makeUnsafe(3)
 *
 * const task = (id: number) =>
 *   semaphore.withPermits(1)(
 *     Effect.gen(function*() {
 *       yield* Effect.yieldNow
 *       return id
 *     })
 *   )
 *
 * // Only 3 tasks can run concurrently
 * const program = Effect.all([
 *   task(1),
 *   task(2),
 *   task(3),
 *   task(4),
 *   task(5)
 * ], { concurrency: "unbounded" })
 *
 * await Effect.runPromise(program) // => [1, 2, 3, 4, 5]
 * ```
 *
 * @stability stable
 * @category constructors
 * @since 4.0.0
 */
export const makeUnsafe = permits => new SemaphoreImpl(permits);
// The first queued request that fits in the free permits. Larger requests
// ahead of it keep their place but are overtaken, as documented on `take`.
const firstAvailableWaiter = self => {
  for (const waiter of self.waiters) {
    if (waiter.permits <= self.free) return waiter;
  }
};
const takePermits = (self, n, onAcquire, restore = identity) => core.withFiber(fiber => {
  if (self.free >= n && firstAvailableWaiter(self) === undefined) {
    self.taken += n;
    return onAcquire();
  }
  // A wake does not reserve permits: the waiter keeps its queue entry until
  // it acquires or is interrupted, and re-checks eligibility when it runs.
  const waiter = {
    permits: n,
    resume: undefined
  };
  const remove = () => {
    if (self.waiters.delete(waiter)) self.releaseUnsafe(fiber, 0);
    return undefined;
  };
  self.waiters.add(waiter);
  internal.onExitUnsafe(fiber, remove);
  const wait = restore(internal.callback(resume => {
    // The fiber may have yielded since the last check, missing a wake.
    if (firstAvailableWaiter(self) === waiter) {
      resume(internal.void);
    } else {
      waiter.resume = resume;
    }
  }));
  const loop = internal.flatMap(wait, () => {
    if (firstAvailableWaiter(self) !== waiter) return loop;
    self.taken += n;
    remove();
    return onAcquire();
  });
  return loop;
});
class SemaphoreImpl {
  waiters = /*#__PURE__*/new Set();
  taken = 0;
  permits;
  constructor(permits) {
    this.permits = permits;
  }
  get free() {
    return this.permits - this.taken;
  }
  take(n) {
    return takePermits(this, n, () => internal.succeed(n));
  }
  takeScoped(n) {
    return internal.uninterruptibleMask(restore => internal.flatMap(internal.scope, scope => takePermits(this, n, () => internal.scopeAddFinalizer(scope, this.release(n)), restore)));
  }
  takeIfAvailable(n) {
    return internal.suspend(() => {
      if (this.free < n) return internal.succeed(false);
      this.taken += n;
      return internal.succeed(true);
    });
  }
  releaseUnsafe(fiber, n) {
    this.taken -= n;
    if (this.waiters.size > 0) {
      fiber.currentDispatcher.scheduleTask(() => {
        // Wake every queued request that fits in the free permits. Waiters
        // that were already woken still count against the budget, since they
        // keep their place until they acquire. Resumed fibers run inline and
        // can add or remove waiters, so iterate over a snapshot.
        let remaining = this.free;
        for (const waiter of Array.from(this.waiters)) {
          if (!this.waiters.has(waiter) || waiter.permits > remaining) continue;
          remaining -= waiter.permits;
          const resume = waiter.resume;
          waiter.resume = undefined;
          resume?.(internal.void);
        }
      }, 0);
    }
    return this.free;
  }
  resize(permits) {
    return core.withFiber(fiber => {
      this.permits = permits;
      if (this.free < 0) return internal.void;
      this.releaseUnsafe(fiber, 0);
      return internal.void;
    });
  }
  release(n) {
    return core.withFiber(fiber => internal.succeed(this.releaseUnsafe(fiber, n)));
  }
  get releaseAll() {
    return core.withFiber(fiber => internal.succeed(this.releaseUnsafe(fiber, this.taken)));
  }
  withPermits(n) {
    return self => internal.uninterruptibleMask(restore => takePermits(this, n, () => internal.onExitPrimitive(restore(self), () => {
      this.releaseUnsafe(internal.getCurrentFiber(), n);
      return undefined;
    }, true), restore));
  }
  withPermit = /*#__PURE__*/this.withPermits(1);
  withPermitsIfAvailable(n) {
    return self => internal.uninterruptibleMask(restore => {
      if (this.free < n) return internal.succeedNone;
      this.taken += n;
      return internal.onExitPrimitive(restore(internal.asSome(self)), () => {
        this.releaseUnsafe(internal.getCurrentFiber(), n);
        return undefined;
      }, true);
    });
  }
}
/**
 * Creates a `Semaphore` initialized with the specified total number of permits.
 *
 * **When to use**
 *
 * Use to create a semaphore inside Effect code for bounding concurrency with
 * automatic or manual permit management.
 *
 * **Example** (Creating a semaphore)
 *
 * ```ts import.meta.vitest
 * import { Effect, Semaphore } from "effect"
 *
 * const program = Effect.gen(function*() {
 *   const semaphore = yield* Semaphore.make(2)
 *
 *   const task = (id: number) =>
 *     semaphore.withPermits(1)(
 *       Effect.gen(function*() {
 *         yield* Effect.yieldNow
 *         return id
 *       })
 *     )
 *
 *   // Run 4 tasks, but only 2 can run concurrently
 *   return yield* Effect.all([task(1), task(2), task(3), task(4)])
 * })
 *
 * await Effect.runPromise(program) // => [1, 2, 3, 4]
 * ```
 *
 * @stability stable
 * @category constructors
 * @since 4.0.0
 */
export const make = permits => internal.sync(() => new SemaphoreImpl(permits));
/**
 * Sets the total number of permits managed by the semaphore.
 *
 * **When to use**
 *
 * Use to change the concurrency limit of an existing semaphore while keeping
 * current acquisitions in place.
 *
 * **Details**
 *
 * Existing acquisitions remain taken after resizing. If the new total is less
 * than the currently taken permit count, new acquisitions wait until enough
 * permits are released.
 *
 * @see {@link make} for creating a semaphore with an initial permit count
 * @see {@link release} for returning permits without changing semaphore capacity
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const resize = /*#__PURE__*/dual(2, (self, permits) => self.resize(permits));
/**
 * Runs an effect with the given number of permits and releases the permits when
 * the effect completes.
 *
 * **When to use**
 *
 * Use to run an effect while holding a specified number of semaphore permits
 * for the duration of that effect.
 *
 * **Details**
 *
 * The effect waits until enough permits are available. Acquired permits are
 * released when the wrapped effect exits.
 *
 * @see {@link withPermit} for acquiring exactly one permit
 * @see {@link withPermitsIfAvailable} for running only when permits are immediately available
 * @see {@link take} for manually acquiring permits
 * @see {@link release} for manually returning permits
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const withPermits = (self, permits, effect) => {
  const withPermits = self.withPermits(permits);
  return effect ? withPermits(effect) : withPermits;
};
/**
 * Runs an effect with a single permit and releases the permit when the effect
 * completes.
 *
 * **When to use**
 *
 * Use to guard an effect with exactly one semaphore permit while automatically
 * releasing that permit when the effect exits.
 *
 * @see {@link withPermits} for acquiring more than one permit
 * @see {@link withPermitsIfAvailable} for running only when permits are immediately available
 * @see {@link take} for manually acquiring permits
 * @see {@link release} for manually returning permits
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const withPermit = (self, effect) => {
  if (!effect) return self.withPermit;
  return self.withPermit(effect);
};
/**
 * Runs an effect only if the specified number of permits are immediately
 * available.
 *
 * **When to use**
 *
 * Use when guarded work should run only if the requested permits are
 * immediately available.
 *
 * **Details**
 *
 * When the permits are unavailable, the effect is not run and the result is
 * `Option.none`. When permits are available, the effect is run, its result is
 * wrapped in `Option.some`, and the acquired permits are released when the
 * effect exits.
 *
 * @see {@link withPermits} for the variant that waits until permits are available
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const withPermitsIfAvailable = (self, permits, effect) => {
  const withPermits = self.withPermitsIfAvailable(permits);
  return effect ? withPermits(effect) : withPermits;
};
/**
 * Acquires the specified number of permits and returns the acquired permit
 * count.
 *
 * **When to use**
 *
 * Use when you need manual permit acquisition for a lower-level protocol with
 * explicit acquisition and release control.
 *
 * **Details**
 *
 * The effect waits until enough permits are available.
 *
 * @see {@link withPermit} for automatically acquiring and releasing one permit around an effect
 * @see {@link withPermits} for automatically acquiring and releasing multiple permits around an effect
 * @see {@link takeIfAvailable} for manually acquiring permits without waiting
 * @see {@link release} for returning manually acquired permits
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const take = /*#__PURE__*/dual(2, (self, permits) => self.take(permits));
/**
 * Acquires the specified number of permits until the current scope closes.
 *
 * **When to use**
 *
 * Use when permits should be held for the lifetime of a scope, such as while a
 * scoped resource is leased.
 *
 * **Details**
 *
 * The effect waits until enough permits are available. The wait can be
 * interrupted unless the caller is uninterruptible. The permits are taken and
 * their release is added to the scope in one uninterruptible step, so acquired
 * permits are always returned when the scope closes.
 *
 * **Example** (Holding a permit for a scope)
 *
 * ```ts import.meta.vitest
 * import { Effect, Semaphore } from "effect"
 *
 * const program = Effect.gen(function*() {
 *   const semaphore = yield* Semaphore.make(1)
 *   yield* Effect.scoped(Semaphore.takeScoped(semaphore, 1))
 *   return yield* Semaphore.takeIfAvailable(semaphore, 1)
 * })
 *
 * Effect.runSync(program) // => true
 * ```
 *
 * @see {@link take} for acquiring permits that are released manually
 * @see {@link withPermits} for holding permits while one effect runs
 *
 * @stability stable
 * @category combinators
 * @since 4.0.7
 */
export const takeScoped = /*#__PURE__*/dual(2, (self, permits) => self.takeScoped(permits));
/**
 * Acquires the specified number of permits only if they are immediately
 * available.
 *
 * **When to use**
 *
 * Use when you need fail-fast manual permit acquisition for a lower-level
 * protocol with explicit acquisition and release control.
 *
 * **Details**
 *
 * If enough permits are available, they are acquired and the effect returns
 * `true`. Otherwise, the effect returns `false` immediately without acquiring
 * any permits.
 *
 * @see {@link take} for the variant that waits until permits are available
 * @see {@link release} for returning manually acquired permits
 * @see {@link withPermitsIfAvailable} for automatic acquisition and release around an effect
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const takeIfAvailable = /*#__PURE__*/dual(2, (self, permits) => self.takeIfAvailable(permits));
/**
 * Releases the specified number of permits and returns the resulting available
 * permits.
 *
 * **When to use**
 *
 * Use when you need to return permits acquired with `take` in a lower-level
 * permit protocol with explicit release control.
 *
 * **Details**
 *
 * Running the effect releases the requested permits, wakes waiting acquirers
 * when permits become available, and returns the current available permit
 * count.
 *
 * **Gotchas**
 *
 * Manual `take` / `release` usage must keep permit counts balanced. Prefer
 * `withPermit` or `withPermits` when the acquisition can be scoped to one
 * effect.
 *
 * @see {@link take} for manually acquiring permits
 * @see {@link releaseAll} for returning every currently taken permit
 * @see {@link withPermits} for automatic acquire and release around an effect
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const release = /*#__PURE__*/dual(2, (self, permits) => self.release(permits));
/**
 * Releases all permits held by this semaphore and returns the resulting
 * available permits.
 *
 * **When to use**
 *
 * Use to return every currently taken permit to a semaphore at once, typically
 * during cleanup of manual `take` / `release` protocols.
 *
 * **Gotchas**
 *
 * This does not stop effects already running with `withPermit` or `withPermits`.
 * Their permits become available immediately, but those effects still release
 * their permits when they finish. The resulting available count can then
 * exceed the semaphore's configured number of permits.
 *
 * @see {@link release} for releasing a known permit count
 * @see {@link withPermits} for automatic acquire and release around an effect
 *
 * @stability stable
 * @category combinators
 * @since 4.0.0
 */
export const releaseAll = self => self.releaseAll;
//# sourceMappingURL=Semaphore.js.map