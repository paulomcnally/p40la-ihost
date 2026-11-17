import * as Cause from "../Cause.js";
import * as Context from "../Context.js";
import * as Effect from "../Effect.js";
import { identity } from "../Function.js";
import * as Layer from "../Layer.js";
import * as Predicate from "../Predicate.js";
import * as Queue from "../Queue.js";
import * as RpcClient from "../rpc/RpcClient.js";
import * as RpcMessage from "../rpc/RpcMessage.js";
import * as RpcSerialization from "../rpc/RpcSerialization.js";
import * as RpcServer from "../rpc/RpcServer.js";
import * as Schema from "../Schema.js";
import * as McpSchema from "./McpSchema.js";
import * as McpServer from "./McpServer.js";
/**
 * Creates an in-memory MCP client for a registration layer.
 *
 * **Details**
 *
 * The server is built from `McpServer.layer` with the supplied `protocols`
 * and `registrations`, over an in-memory `RpcServer.Protocol`. The client
 * speaks `options.protocolVersion`, by default the first adapter's version:
 * stateful versions `initialize` and send `notifications/initialized` before
 * `makeClient` returns; the stateless version attaches request metadata to
 * every request instead.
 *
 * Because requests reach handlers through the server runner, the transport
 * built-ins are discharged the same way as on stdio: `McpRequestContext` on
 * every request, and `McpServerClient` only when every adapter in `protocols`
 * is stateful (see `McpServer.ProvidedServices`). Any other `McpRequest`
 * marker in `registrations` stays in the requirements and must be met by MCP
 * middleware; providing the service directly does not discharge it.
 *
 * The client and server live in the surrounding `Scope`.
 *
 * **Example** (Reading a resource in a test)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { McpProtocol, McpServer, McpTest } from "effect/ai"
 *
 * const registrations = McpServer.resource({
 *   uri: "test://greeting",
 *   name: "greeting",
 *   content: Effect.succeed("hello")
 * })
 *
 * const program = Effect.gen(function*() {
 *   const client = yield* McpTest.makeClient(registrations, {
 *     protocols: [McpProtocol.v2025_11_25]
 *   })
 *   const result = yield* client.readResource({ uri: "test://greeting" })
 *   return result.contents.map((content) => content.uri)
 * }).pipe(Effect.scoped)
 *
 * const actual = await Effect.runPromise(program)
 * actual // => ["test://greeting"]
 * ```
 *
 * @see {@link Client} for the request methods and notifications
 * @see {@link McpServer.middleware} for discharging request-level dependencies
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const makeClient = /*#__PURE__*/Effect.fnUntraced(function* (registrations, options) {
  const selected = options.protocolVersion === undefined ? options.protocols[0] : options.protocols.find(protocol => protocol.protocolVersion === options.protocolVersion);
  if (selected === undefined) {
    return yield* new Cause.IllegalArgumentError(`MCP protocol ${options.protocolVersion} is not in the configured protocols`);
  }
  const clientInfo = options.clientInfo ?? defaultClientInfo;
  const clientCapabilities = options.clientCapabilities ?? new McpSchema.ClientCapabilities({});
  // Each direction carries one serialized JSON-RPC message per element, so
  // requests and responses round-trip through the same JSON-RPC encoding as
  // the stdio and HTTP transports.
  const toServer = yield* Queue.unbounded();
  const toClient = yield* Queue.unbounded();
  const notifications = yield* Queue.unbounded();
  const serverProtocol = RpcServer.Protocol.make(Effect.fnUntraced(function* (writeRequest) {
    const parser = serialization.makeUnsafe();
    yield* Queue.take(toServer).pipe(Effect.flatMap(frame => Effect.forEach(parser.decode(frame), message => writeRequest(0, message), {
      discard: true
    })), Effect.forever, Effect.forkScoped);
    return {
      disconnects: yield* Queue.make(),
      send(_clientId, response) {
        const encoded = parser.encode(response);
        return typeof encoded === "string" ? Effect.asVoid(Queue.offer(toClient, encoded)) : Effect.void;
      },
      end: () => Effect.void,
      clientIds: Effect.succeed(new Set([0])),
      initialMessage: Effect.succeedNone,
      supportsAck: false,
      supportsTransferables: false,
      supportsSpanPropagation: false,
      supportsNotifications: true,
      codecFor: serialization.codecFor
    };
  }));
  const context = yield* Layer.build(McpServer.layer({
    name: options.name ?? "McpTest",
    version: options.version ?? "1.0.0",
    instructions: options.instructions,
    extensions: options.extensions,
    protocols: options.protocols,
    registrations
  }).pipe(Layer.provide(Layer.effect(RpcServer.Protocol, serverProtocol))));
  const server = Context.get(context, McpServer.McpServer);
  const write = message => Effect.asVoid(Queue.offer(toServer, JSON.stringify(message)));
  // Stateless revisions carry the client profile on every request instead of
  // negotiating it with `initialize`.
  const requestMetadata = selected.runtime._tag === "Stateless" ? {
    "io.modelcontextprotocol/protocolVersion": selected.protocolVersion,
    "io.modelcontextprotocol/clientCapabilities": encodeClientCapabilities(clientCapabilities),
    "io.modelcontextprotocol/clientInfo": encodeImplementation(clientInfo)
  } : undefined;
  const params = payload => {
    if (requestMetadata === undefined) {
      return payload === undefined || payload === null ? {} : {
        params: payload
      };
    }
    const fields = Predicate.isReadonlyObject(payload) ? payload : {};
    return {
      params: {
        ...fields,
        _meta: {
          ...requestMetadata,
          ...(Predicate.isReadonlyObject(fields._meta) ? fields._meta : {})
        }
      }
    };
  };
  const protocol = yield* RpcClient.Protocol.make(Effect.fnUntraced(function* (writeResponse) {
    const parser = serialization.makeUnsafe();
    let clientId = 0;
    const route = message => {
      if (message._tag !== "Request") {
        return writeResponse(clientId, message);
      }
      if (message.isNotification === true) {
        return Effect.asVoid(Queue.offer(notifications, {
          _tag: "Notification",
          method: message.tag,
          payload: message.payload ?? undefined
        }));
      }
      // Server-to-client requests are not supported, so reject them instead
      // of leaving the handler waiting for a reply.
      return write({
        jsonrpc: "2.0",
        id: message.id,
        error: {
          code: McpSchema.METHOD_NOT_FOUND_ERROR_CODE,
          message: `Method not found: ${message.tag}`
        }
      });
    };
    yield* Queue.take(toClient).pipe(Effect.flatMap(frame => Effect.forEach(parser.decode(frame), route, {
      discard: true
    })), Effect.forever, Effect.forkScoped);
    return {
      send(id, request) {
        clientId = id;
        switch (request._tag) {
          case "Request":
            return write({
              jsonrpc: "2.0",
              id: request.id,
              method: request.tag,
              ...params(request.payload)
            });
          case "Interrupt":
            return write({
              jsonrpc: "2.0",
              method: "notifications/cancelled",
              ...params({
                requestId: request.requestId
              })
            });
          default:
            // Effect RPC control messages are not part of the MCP protocol.
            return Effect.void;
        }
      },
      supportsAck: false,
      supportsTransferables: false,
      codecFor: serialization.codecFor
    };
  }));
  let nextRequestId = 1;
  const rpc = yield* RpcClient.make(McpSchema.ClientRequestRpcs, {
    spanPrefix: "McpTest",
    generateRequestId: () => RpcMessage.RequestId(nextRequestId++)
  }).pipe(Effect.provideService(RpcClient.Protocol, protocol));
  let protocolVersion = selected.protocolVersion;
  if (selected.runtime._tag === "Stateful") {
    const result = yield* rpc.initialize({
      protocolVersion: selected.protocolVersion,
      capabilities: clientCapabilities,
      clientInfo
    }).pipe(Effect.orDie);
    protocolVersion = result.protocolVersion;
    yield* write({
      jsonrpc: "2.0",
      method: "notifications/initialized"
    });
  }
  return identity({
    protocolVersion,
    server,
    notifications,
    ping: Effect.asVoid(orDieClientError(rpc.ping(undefined))),
    listTools: request => orDieClientError(rpc["tools/list"](request)),
    callTool: request => orDieClientError(rpc["tools/call"](request)),
    listResources: request => orDieClientError(rpc["resources/list"](request)),
    listResourceTemplates: request => orDieClientError(rpc["resources/templates/list"](request)),
    readResource: request => orDieClientError(rpc["resources/read"](request)),
    subscribe: request => Effect.asVoid(orDieClientError(rpc["resources/subscribe"](request))),
    unsubscribe: request => Effect.asVoid(orDieClientError(rpc["resources/unsubscribe"](request))),
    listPrompts: request => orDieClientError(rpc["prompts/list"](request)),
    getPrompt: request => orDieClientError(rpc["prompts/get"](request)),
    complete: request => orDieClientError(rpc["completion/complete"](request)),
    setLevel: request => Effect.asVoid(orDieClientError(rpc["logging/setLevel"](request)))
  });
});
const serialization = /*#__PURE__*/RpcSerialization.jsonRpc();
const defaultClientInfo = {
  name: "McpTest",
  version: "1.0.0"
};
const encodeClientCapabilities = /*#__PURE__*/Schema.encodeSync(/*#__PURE__*/Schema.toCodecJson(McpSchema.ClientCapabilities));
const encodeImplementation = /*#__PURE__*/Schema.encodeSync(/*#__PURE__*/Schema.toCodecJson(McpSchema.Implementation));
// The in-memory transport never fails to send, so a client error is a defect.
const orDieClientError = effect => Effect.catchIf(effect, error => Predicate.isTagged(error, "RpcClientError"), Effect.die);
//# sourceMappingURL=McpTest.js.map