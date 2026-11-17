import * as Effect from "../Effect.ts";
import * as Sse from "../encoding/Sse.ts";
import type * as Headers from "../http/Headers.ts";
import * as HttpClient from "../http/HttpClient.ts";
import * as HttpClientError from "../http/HttpClientError.ts";
import type * as HttpClientResponse from "../http/HttpClientResponse.ts";
import * as Schema from "../Schema.ts";
import type * as Socket from "../socket/Socket.ts";
import * as Stream from "../Stream.ts";
import type { Simplify } from "../Types.ts";
import * as HttpApi from "./HttpApi.ts";
import * as HttpApiEndpoint from "./HttpApiEndpoint.ts";
import type * as HttpApiGroup from "./HttpApiGroup.ts";
import type * as HttpApiMiddleware from "./HttpApiMiddleware.ts";
import * as HttpApiSchema from "./HttpApiSchema.ts";
/**
 * The type-safe client shape generated from HTTP API groups, with non-top-level
 * groups exposed as nested objects and top-level endpoints exposed as methods.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export type Client<Groups extends HttpApiGroup.Constraint, E = never, R = never> = Simplify<{
    readonly [Group in Extract<Groups, {
        readonly topLevel: false;
    }> as HttpApiGroup.Identifier<Group>]: Client.Group<Group, E, R>;
} & Client.TopLevelMethods<Groups, E, R>>;
/**
 * Derives the typed client interface for an `HttpApi`, preserving any additional
 * client error and service requirements supplied by the caller.
 *
 * @stability unstable
 * @category utility types
 * @since 4.0.0
 */
export type ForApi<Api extends HttpApi.Constraint, E = never, R = never> = Api extends HttpApi.HttpApi<infer _Id, infer Groups> ? Client<Groups, E, R> : never;
/**
 * Computes the decoded success value a client receives for an endpoint
 * success schema, including headers, streaming Server-Sent Events, and
 * streaming byte responses.
 *
 * @stability unstable
 * @category utility types
 * @since 4.0.0
 */
export type SuccessType<S> = S extends HttpApiSchema.WithHeaders<infer _Inner, infer _Headers> ? HttpApiSchema.withHeaders<SuccessType<_Inner>, _Headers["Type"]> : S extends HttpApiSchema.StreamSse<infer _Events, infer _Error, infer _Value> ? Stream.Stream<_Value, _Error["Type"] | HttpClientError.HttpClientError | Schema.SchemaError | Sse.Retry | Sse.SseError, never> : S extends HttpApiSchema.StreamUint8Array ? Stream.Stream<Uint8Array, HttpClientError.HttpClientError, never> : S extends HttpApiSchema.StreamNdjson<infer _S> ? Stream.Stream<_S["Type"], HttpClientError.HttpClientError | Schema.SchemaError, never> : S extends HttpApiSchema.Upgrade<infer _Headers> ? [_Headers] extends [never] ? Upgraded : Upgraded<_Headers["Type"]> : S extends Schema.Constraint ? S["Type"] : never;
/**
 * Runtime brand key used to mark `Upgraded` values.
 *
 * @stability unstable
 * @category type IDs
 * @since 4.1.0
 */
export declare const UpgradedTypeId = "~effect/http-api/HttpApiClient/Upgraded";
/**
 * Type-level brand identifier used by `Upgraded` values.
 *
 * @stability unstable
 * @category type IDs
 * @since 4.1.0
 */
export type UpgradedTypeId = typeof UpgradedTypeId;
/**
 * The value a client method returns for an endpoint declared with
 * `HttpApiSchema.Upgrade`: the headers of the `101 Switching Protocols`
 * response and the upgraded connection.
 *
 * **Details**
 *
 * `headers` is decoded with the upgrade's `headers` schema when it declares
 * one. See `HttpClientResponse.upgrade` for how the connection behaves and
 * when it closes.
 *
 * When the endpoint also declares ordinary success responses, the method
 * returns either an `Upgraded` value or the decoded ordinary success. Use
 * {@link isUpgraded} to tell them apart.
 *
 * @stability unstable
 * @category models
 * @since 4.0.6
 */
export interface Upgraded<Headers = Headers.Headers> {
    readonly [UpgradedTypeId]: UpgradedTypeId;
    readonly headers: Headers;
    readonly socket: Socket.Socket;
}
/**
 * Returns `true` when a client method result is an upgraded connection.
 *
 * **Details**
 *
 * An endpoint may declare `HttpApiSchema.Upgrade` alongside ordinary success
 * responses, because a server can decline an upgrade (RFC 9110 §7.8). The
 * client method then returns either an `Upgraded` value or the decoded
 * ordinary success, depending on the response status. This predicate narrows
 * that union.
 *
 * **Example** (Handling a declined upgrade)
 *
 * ```ts
 * import { Effect, Schema } from "effect"
 * import { HttpApi, HttpApiClient, HttpApiEndpoint, HttpApiGroup, HttpApiSchema } from "effect/http-api"
 *
 * const Api = HttpApi.make("docker").add(
 *   HttpApiGroup.make("exec").add(
 *     HttpApiEndpoint.post("start", "/exec/:id/start", {
 *       params: { id: Schema.String },
 *       payload: Schema.Struct({ Detach: Schema.Boolean }),
 *       success: [HttpApiSchema.Upgrade({ protocol: "tcp" }), HttpApiSchema.Empty(200)]
 *     })
 *   )
 * )
 *
 * const program = Effect.gen(function*() {
 *   const client = yield* HttpApiClient.make(Api)
 *   const result = yield* client.exec.start({ params: { id: "abc" }, payload: { Detach: false } })
 *   if (HttpApiClient.isUpgraded(result)) {
 *     const writer = yield* result.socket.writer
 *     yield* writer.write("ls\n")
 *   }
 * })
 * ```
 *
 * @stability unstable
 * @category guards
 * @since 4.1.0
 */
export declare const isUpgraded: (u: unknown) => u is Upgraded<unknown>;
type SuccessDecodingServices<S> = S extends HttpApiSchema.StreamSse<infer _Events, infer _Error, infer _Value> ? _Events["DecodingServices"] | _Error["DecodingServices"] : S extends HttpApiSchema.StreamUint8Array ? never : S extends HttpApiSchema.StreamNdjson<infer _S> ? _S["DecodingServices"] : S extends HttpApiSchema.Upgrade<infer _Headers> ? _Headers["DecodingServices"] : S extends Schema.Constraint ? S["DecodingServices"] : never;
type PayloadServicesConstraint<Payload> = [
    Extract<Payload, HttpApiSchema.StreamUint8Array | HttpApiSchema.StreamNdjson<any>>
] extends [never] ? never : unknown;
/**
 * Helper types used to describe generated HTTP API clients, including endpoint
 * methods, response modes, and grouped client shapes.
 *
 * @stability unstable
 * @since 4.0.0
 */
export declare namespace Client {
    /**
     * The response mode accepted by generated client methods, controlling whether a
     * call returns the decoded success value, the raw response, or both.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    type ResponseMode = HttpApiEndpoint.ClientResponseMode;
    /**
     * Computes the value returned by a client method for a success type and response
     * mode.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    type Response<Success, Mode extends ResponseMode> = Mode extends "decoded-and-response" ? [Success, HttpClientResponse.HttpClientResponse] : Mode extends "response-only" ? HttpClientResponse.HttpClientResponse : Success;
    type GroupByEndpoint<Group extends HttpApiGroup.Constraint, E, R> = Group["endpoints"] extends infer Endpoints extends Readonly<Record<string, HttpApiEndpoint.ConstraintRequest>> ? {
        readonly [Identifier in keyof Endpoints]: Method<Endpoints[Identifier], E, R>;
    } : {};
    /**
     * The client object for one API group, mapping each endpoint identifier in that
     * group to its typed client method.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    type Group<Group extends HttpApiGroup.Constraint, E, R> = GroupByEndpoint<Group, E, R>;
    type MethodReturn<Endpoint extends HttpApiEndpoint.ConstraintRequest, E, R, Mode extends ResponseMode, PayloadR> = Effect.Effect<Response<SuccessType<Endpoint["~Success"]>, Mode>, HttpApiMiddleware.Error<Endpoint["~Middleware"]> | HttpApiMiddleware.ClientError<Endpoint["~Middleware"]> | E | HttpClientError.HttpClientError | ([Mode] extends ["response-only"] ? never : Endpoint["~Error"]["Type"] | Schema.SchemaError), R | PayloadR | Endpoint["~Params"]["EncodingServices"] | Endpoint["~Query"]["EncodingServices"] | Endpoint["~Payload"]["EncodingServices"] | Endpoint["~Headers"]["EncodingServices"] | ([Mode] extends ["response-only"] ? never : SuccessDecodingServices<Endpoint["~Success"]> | Endpoint["~Error"]["DecodingServices"])>;
    /**
     * The typed function generated for an endpoint, accepting the endpoint request
     * shape, including optional per-call SSE decoding options. The returned effect's success,
     * error, and service channels reflect the endpoint schemas, middleware, and
     * selected response mode.
     *
     * **Details**
     *
     * The services of a streaming `payload` are inferred and added to the
     * returned effect.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    type Method<Endpoint extends HttpApiEndpoint.ConstraintRequest, E, R> = <Mode extends ResponseMode = "decoded-only", PayloadR extends PayloadServicesConstraint<Endpoint["~Payload"]> = never>(request: Simplify<HttpApiEndpoint.ClientRequest<Endpoint["~Params"], Endpoint["~Query"], Endpoint["~Payload"], Endpoint["~Headers"], Mode, PayloadR>>) => MethodReturn<Endpoint, E, R, Mode, NoInfer<PayloadR>>;
    /**
     * Extracts client methods for endpoints in top-level groups so they can be exposed
     * directly on the generated client object.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    type TopLevelMethods<Groups extends HttpApiGroup.Constraint, E, R> = {
        readonly [Endpoint in Extract<HttpApiGroup.Endpoints<Extract<Groups, {
            readonly topLevel: true;
        }>>, HttpApiEndpoint.ConstraintRequest> as Endpoint["identifier"]]: Method<Endpoint, E, R>;
    };
}
type UrlBuilderRequestPart<Key extends string, Value> = [Value] extends [never] ? {} : {
    readonly [K in Key]: Value;
};
type UrlBuilderRequest<Endpoint extends HttpApiEndpoint.Constraint, Params = HttpApiEndpoint.Params<Endpoint>["Type"], Query = HttpApiEndpoint.Query<Endpoint>["Type"]> = (UrlBuilderRequestPart<"params", Params> & UrlBuilderRequestPart<"query", Query>) extends infer Request ? keyof Request extends never ? void | undefined : Request : never;
type UrlBuilderArgs<Request> = [Request] extends [void | undefined] ? [request?: Request] : [request: Request];
/**
 * The type-safe URL builder shape for an HTTP API, mirroring the generated client
 * layout while returning URL strings instead of executing requests.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export type UrlBuilder<Api extends HttpApi.Constraint> = Api extends HttpApi.HttpApi<infer _ApiId, infer Groups> ? [
    Extract<Groups, {
        readonly topLevel: true;
    }>
] extends [never] ? UrlBuilderGroups<Groups> : [Extract<Groups, {
    readonly topLevel: false;
}>] extends [never] ? UrlBuilderTopLevelMethods<Groups> : Simplify<UrlBuilderGroups<Groups> & UrlBuilderTopLevelMethods<Groups>> : never;
type UrlBuilderGroups<Groups extends HttpApiGroup.Constraint> = {
    readonly [Group in Extract<Groups, {
        readonly topLevel: false;
    }> as HttpApiGroup.Identifier<Group>]: UrlBuilderGroup<HttpApiGroup.Endpoints<Group>>;
};
type UrlBuilderGroup<Endpoints extends HttpApiEndpoint.Constraint> = {
    readonly [Endpoint in Endpoints as HttpApiEndpoint.Identifier<Endpoint>]: UrlBuilderMethod<Endpoint>;
};
type UrlBuilderMethod<Endpoint extends HttpApiEndpoint.Constraint> = (...args: UrlBuilderArgs<UrlBuilderRequest<Endpoint>>) => string;
type UrlBuilderTopLevelMethods<Groups extends HttpApiGroup.Constraint> = {
    readonly [Endpoint in HttpApiGroup.Endpoints<Extract<Groups, {
        readonly topLevel: true;
    }>> as HttpApiEndpoint.Identifier<Endpoint>]: UrlBuilderMethod<Endpoint>;
};
/**
 * Constructs a type-safe client for an HTTP API using the `HttpClient` service,
 * endpoint schemas, middleware, and optional client or response transformations.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const make: <ApiId extends string, Groups extends HttpApiGroup.Constraint>(api: HttpApi.HttpApi<ApiId, Groups>, options?: {
    readonly transformClient?: ((client: HttpClient.HttpClient) => HttpClient.HttpClient) | undefined;
    readonly transformResponse?: ((effect: Effect.Effect<unknown, unknown, unknown>) => Effect.Effect<unknown, unknown, unknown>) | undefined;
    readonly baseUrl?: URL | string | undefined;
}) => Effect.Effect<Client<Groups>, never, HttpClient.HttpClient | HttpApiGroup.MiddlewareClient<Groups>>;
/**
 * Constructs a type-safe client for an HTTP API from the supplied `HttpClient`,
 * using the API metadata to encode requests, execute middleware, and decode
 * responses.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeWith: <ApiId extends string, Groups extends HttpApiGroup.Constraint, E, R>(api: HttpApi.HttpApi<ApiId, Groups>, options: {
    readonly httpClient: HttpClient.HttpClient.With<E, R>;
    readonly transformResponse?: ((effect: Effect.Effect<unknown, unknown, unknown>) => Effect.Effect<unknown, unknown, unknown>) | undefined;
    readonly baseUrl?: URL | string | undefined;
}) => Effect.Effect<Client<Groups, Exclude<E, HttpClientError.HttpClientError>, R>, never, HttpApiGroup.MiddlewareClient<Groups>>;
/**
 * Builds a typed client object for a single API group from the supplied
 * `HttpClient`, filtering the API to that group.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const group: <ApiId extends string, Groups extends HttpApiGroup.Constraint, const GroupIdentifier extends HttpApiGroup.Identifier<Groups>, E, R>(api: HttpApi.HttpApi<ApiId, Groups>, options: {
    readonly group: GroupIdentifier;
    readonly httpClient: HttpClient.HttpClient.With<E, R>;
    readonly transformResponse?: ((effect: Effect.Effect<unknown, unknown, unknown>) => Effect.Effect<unknown, unknown, unknown>) | undefined;
    readonly baseUrl?: URL | string | undefined;
}) => Effect.Effect<Client.Group<HttpApiGroup.WithIdentifier<Groups, GroupIdentifier>, E, R>, never, HttpApiGroup.MiddlewareClient<HttpApiGroup.WithIdentifier<Groups, GroupIdentifier>>>;
type EndpointReturn<Groups extends HttpApiGroup.Constraint, GroupIdentifier extends HttpApiGroup.Identifier<Groups>, EndpointIdentifier extends HttpApiGroup.EndpointsWithIdentifier<Groups, GroupIdentifier>["identifier"], E, R, Endpoint extends HttpApiEndpoint.ConstraintRequest = Extract<HttpApiEndpoint.WithIdentifier<HttpApiGroup.EndpointsWithIdentifier<Groups, GroupIdentifier>, EndpointIdentifier>, HttpApiEndpoint.ConstraintRequest>> = Effect.Effect<Client.Method<Endpoint, E, R>, never, HttpApiEndpoint.MiddlewareClient<Endpoint>>;
/**
 * Builds the typed client method for one endpoint in one API group, using the
 * supplied `HttpClient` and endpoint metadata.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const endpoint: <ApiId extends string, Groups extends HttpApiGroup.Constraint, const GroupIdentifier extends HttpApiGroup.Identifier<Groups>, const EndpointIdentifier extends HttpApiGroup.EndpointsWithIdentifier<Groups, GroupIdentifier>["identifier"], E, R>(api: HttpApi.HttpApi<ApiId, Groups>, options: {
    readonly group: GroupIdentifier;
    readonly endpoint: EndpointIdentifier;
    readonly httpClient: HttpClient.HttpClient.With<E, R>;
    readonly transformClient?: ((client: HttpClient.HttpClient.With<E, R>) => HttpClient.HttpClient.With<E, R>) | undefined;
    readonly transformResponse?: ((effect: Effect.Effect<unknown, unknown, unknown>) => Effect.Effect<unknown, unknown, unknown>) | undefined;
    readonly baseUrl?: URL | string | undefined;
}) => EndpointReturn<Groups, GroupIdentifier, EndpointIdentifier, E, R>;
type DecodeResponse<Groups extends HttpApiGroup.Constraint, GroupIdentifier extends HttpApiGroup.Identifier<Groups>, EndpointIdentifier extends HttpApiGroup.EndpointsWithIdentifier<Groups, GroupIdentifier>["identifier"], Endpoint extends HttpApiEndpoint.ConstraintRequest = Extract<HttpApiEndpoint.WithIdentifier<HttpApiGroup.EndpointsWithIdentifier<Groups, GroupIdentifier>, EndpointIdentifier>, HttpApiEndpoint.ConstraintRequest>> = (response: HttpClientResponse.HttpClientResponse, options?: {
    readonly sseOptions?: Sse.DecodeOptions | undefined;
}) => Effect.Effect<SuccessType<Endpoint["~Success"]>, HttpApiEndpoint.Errors<Endpoint> | HttpClientError.HttpClientError | Schema.SchemaError, SuccessDecodingServices<Endpoint["~Success"]> | HttpApiEndpoint.ErrorServicesDecode<Endpoint>>;
/**
 * Builds a decoder for raw responses of one endpoint in one API group, using
 * the same success and error decoding as a generated client method in
 * `"decoded-only"` mode.
 *
 * **When to use**
 *
 * Use with `responseMode: "response-only"` when the response head and body
 * arrive at different times, such as long polling, and the body should be
 * decoded later.
 *
 * **Details**
 *
 * The response does not need to come from a generated client. Declared error
 * responses fail with the decoded error, and undeclared statuses fail with an
 * `HttpClientError` whose reason is a `DecodeError`. Parse options are resolved
 * from the API, group and endpoint annotations. The body is only read when the
 * returned effect runs, so it must run inside the scope that owns the response.
 * Middleware and `transformResponse` are not applied.
 *
 * Building the decoder walks the API declaration, so build it once per
 * endpoint and reuse it.
 *
 * **Example** (Decoding a response body after the head arrives)
 *
 * ```ts import.meta.vitest
 * import { Effect, Schema } from "effect"
 * import { HttpApi, HttpApiClient, HttpApiEndpoint, HttpApiGroup } from "effect/http-api"
 *
 * const Api = HttpApi.make("Api").add(
 *   HttpApiGroup.make("containers").add(
 *     HttpApiEndpoint.post("wait", "/containers/:id/wait", {
 *       params: { id: Schema.String },
 *       success: Schema.Struct({ StatusCode: Schema.Number })
 *     })
 *   )
 * )
 *
 * const decodeWait = HttpApiClient.decodeResponse(Api, {
 *   group: "containers",
 *   endpoint: "wait"
 * })
 *
 * const program = Effect.gen(function*() {
 *   const client = yield* HttpApiClient.make(Api)
 *   // Succeeds as soon as the response head arrives
 *   const response = yield* client.containers.wait({
 *     params: { id: "my-container" },
 *     responseMode: "response-only"
 *   })
 *   // Waits for the body, then decodes it with the endpoint schemas
 *   return yield* decodeWait(response)
 * })
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.7
 */
export declare const decodeResponse: <ApiId extends string, Groups extends HttpApiGroup.Constraint, const GroupIdentifier extends HttpApiGroup.Identifier<Groups>, const EndpointIdentifier extends HttpApiGroup.EndpointsWithIdentifier<Groups, GroupIdentifier>["identifier"]>(api: HttpApi.HttpApi<ApiId, Groups>, options: {
    readonly group: GroupIdentifier;
    readonly endpoint: EndpointIdentifier;
}) => DecodeResponse<Groups, GroupIdentifier, EndpointIdentifier>;
/**
 * Creates a type-safe URL builder that mirrors `HttpApiClient.make`.
 *
 * **Example** (Building typed URLs)
 *
 * ```ts import.meta.vitest
 * import { Schema } from "effect"
 * import { HttpApi, HttpApiClient, HttpApiEndpoint, HttpApiGroup } from "effect/http-api"
 *
 * const Api = HttpApi.make("Api").add(
 *   HttpApiGroup.make("users").add(
 *     HttpApiEndpoint.get("getUser", "/users/:id", {
 *       params: { id: Schema.String }
 *     })
 *   )
 * )
 *
 * const buildUrl = HttpApiClient.urlBuilder(Api, {
 *   baseUrl: "https://api.example.com"
 * })
 *
 * buildUrl.users.getUser({
 *   params: { id: "123" }
 * }) // => "https://api.example.com/users/123"
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const urlBuilder: <Api extends HttpApi.Constraint>(api: Api, options?: {
    readonly baseUrl?: URL | string | undefined;
}) => UrlBuilder<Api>;
export {};
//# sourceMappingURL=HttpApiClient.d.ts.map