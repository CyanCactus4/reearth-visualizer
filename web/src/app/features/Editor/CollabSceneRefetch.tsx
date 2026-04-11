import { useApolloClient } from "@apollo/client/react";
import { useCollab } from "@reearth/services/collab";
import { GET_SCENE } from "@reearth/services/gql/queries/scene";
import { useLang } from "@reearth/services/i18n/hooks";
import { FC, useEffect, useRef } from "react";

/**
 * When collab `applied` messages carry a new sceneRev, refetch GetScene so peers
 * pick up widget/scene changes without a full reload (PLAN.md phase 2).
 */
const CollabSceneRefetch: FC<{ sceneId: string }> = ({ sceneId }) => {
  const client = useApolloClient();
  const collab = useCollab();
  const lang = useLang();
  const prevRev = useRef<number | undefined>(undefined);

  useEffect(() => {
    prevRev.current = undefined;
  }, [sceneId]);

  useEffect(() => {
    const rev = collab?.remoteSceneRev;
    if (rev == null) return;
    if (rev === prevRev.current) return;
    prevRev.current = rev;
    if (rev === 0) return;
    void client.query({
      query: GET_SCENE,
      variables: { sceneId, lang },
      fetchPolicy: "network-only"
    });
  }, [client, collab?.remoteSceneRev, sceneId, lang]);

  return null;
};

export default CollabSceneRefetch;
