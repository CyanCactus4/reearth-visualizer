import { ApolloLink, split } from "@apollo/client";
import { GraphQLWsLink } from "@apollo/client/link/subscriptions";
import { getMainDefinition } from "@apollo/client/utilities";
import { createClient } from "graphql-ws";

/** Map HTTP(S) GraphQL URL to WebSocket URL for graphql-ws (gqlgen supports graphql-transport-ws). */
export function graphqlSubscriptionWsUrl(httpGraphqlUrl: string): string {
  if (httpGraphqlUrl.startsWith("https://")) {
    return httpGraphqlUrl.replace("https://", "wss://");
  }
  if (httpGraphqlUrl.startsWith("http://")) {
    return httpGraphqlUrl.replace("http://", "ws://");
  }
  const proto =
    typeof globalThis !== "undefined" &&
    globalThis.location?.protocol === "https:"
      ? "wss"
      : "ws";
  const host =
    typeof globalThis !== "undefined" ? globalThis.location?.host ?? "" : "";
  const path = httpGraphqlUrl.startsWith("/")
    ? httpGraphqlUrl
    : `/${httpGraphqlUrl}`;
  return `${proto}://${host}${path}`;
}

export function buildSubscriptionSplitLink(opts: {
  getAccessToken: () => Promise<string | null>;
  httpGraphqlEndpoint: string;
  /** Links before the split: typically task + error + sentry. */
  leadingLinks: ApolloLink[];
  /** HTTP branch after split: auth + lang + upload. */
  httpTail: ApolloLink;
}): ApolloLink {
  const wsUrl = graphqlSubscriptionWsUrl(opts.httpGraphqlEndpoint);
  const wsLink = new GraphQLWsLink(
    createClient({
      url: wsUrl,
      connectionParams: async () => {
        const token = await opts.getAccessToken();
        return token ? { Authorization: `Bearer ${token}` } : {};
      },
      lazy: true
    })
  );
  return ApolloLink.from([
    ...opts.leadingLinks,
    split(
      ({ query }) => {
        const def = getMainDefinition(query);
        return (
          def.kind === "OperationDefinition" &&
          def.operation === "subscription"
        );
      },
      wsLink,
      opts.httpTail
    )
  ]);
}
