import { useApolloClient } from "@apollo/client/react";
import {
  COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES,
  useCollab
} from "@reearth/services/collab";
import { GET_SCENE } from "@reearth/services/gql/queries/scene";
import { useLang } from "@reearth/services/i18n/hooks";
import { FC, useEffect, useRef } from "react";

/**
 * When collab `applied` messages carry a new sceneRev, refetch GetScene so peers
 * pick up widget/scene changes without a full reload (PLAN.md phase 2).
 * On `apply` failure errors, refetch so optimistic UI realigns (PLAN.md phase 3).
 */
const CollabSceneRefetch: FC<{ sceneId: string }> = ({ sceneId }) => {
  const client = useApolloClient();
  const collab = useCollab();
  const lang = useLang();
  const prevRev = useRef<number | undefined>(undefined);
  const lastApplyErrRef = useRef<string>("");

  useEffect(() => {
    prevRev.current = undefined;
    lastApplyErrRef.current = "";
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

  useEffect(() => {
    const msg = collab?.lastMessage;
    if (msg?.t !== "error") return;
    const d = msg.d as { code?: string; message?: string } | undefined;
    const code = typeof d?.code === "string" ? d.code : "";
    if (!COLLAB_APPLY_FAILURE_SCENE_REFETCH_CODES.has(code)) return;
    const fingerprint = `${code}:${d?.message ?? ""}`;
    if (fingerprint === lastApplyErrRef.current) return;
    lastApplyErrRef.current = fingerprint;
    void client.query({
      query: GET_SCENE,
      variables: { sceneId, lang },
      fetchPolicy: "network-only"
    });
  }, [client, collab?.lastMessage, sceneId, lang]);

  return null;
};

export default CollabSceneRefetch;
