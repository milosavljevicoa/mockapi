This plan replaces the channel-level response override (previously implemented) with a
per-path response rule system. Rules are keyed by (channelSlug, method, path) so that
different paths within the same channel can return independent responses.

All channel-level response fields and endpoints introduced by the prior implementation
must be reverted as part of this work.

---

1. store/models.go — Revert: remove ResponseStatus and ResponseBody from Channel.
   No new exported types are needed; the rule store is internal to the store package.

2. store/store.go — Replace channel response methods with a rules map:
   - Remove GetChannelResponse and UpdateChannelResponse.
   - Remove ResponseStatus/ResponseBody defaults from CreateChannel and GetOrCreateChannel.
   - Add an unexported responseRule struct and a rules map[string]*responseRule field to Store.
   - Add GetResponseRule(slug, method, path string) (status int, body []byte) —
     returns (200, []byte("{}")) when no rule exists for the key.
   - Add SetResponseRule(slug, method, path string, status int, body []byte) error —
     returns an error if the channel does not exist, upserts otherwise.

3. handler/api.go — Revert channel changes; add request-level response to the API:
   - Revert channelJSON and toChannelJSON to their original forms (remove ResponseStatus,
     ResponseBody fields and the PATCH /_/api/channels/{slug} handler).
   - Add ResponseStatus int and ResponseBody json.RawMessage to requestJSON.
   - Update toRequestJSON to accept (r *store.Request, respStatus int, respBody []byte)
     and populate the new fields; update all call sites to supply the rule values.
   - Add PATCH /_/api/channels/{slug}/requests/{id}:
       - Look up the request to get its Method and Path.
       - Decode body: response_status (*int, required) and response_body (json.RawMessage, required).
       - Validate presence, range 100–599, and json.Valid.
       - Call st.SetResponseRule(slug, method, path, status, body).
       - Return the updated requestJSON with 200 OK.

4. handler/capture.go — Use rule lookup instead of channel-level response:
   - Replace st.GetChannelResponse(slug) with
     st.GetResponseRule(slug, r.Method, r.URL.Path).
   - Content-Type and WriteHeader ordering unchanged.

5. ui/static/index.html — Move response editing from channel sidebar to request detail:
   - Revert: remove .status-badge CSS, the status badge from renderChannels(), the
     Edit Response modal HTML, the "Edit response" context menu item, and all
     associated JS (openEditResponseModal, closeEditResponseModal, save handler).
   - Add: in the request detail panel (renderDetail), add a "Response" section below
     the body section showing the configured response_status and response_body for
     that request's method+path.
   - Add an "Edit response" button inside the detail panel that opens a modal
     pre-filled with the current response_status and response_body.
   - The modal sends PATCH /_/api/channels/{slug}/requests/{id}; on success refresh
     the request detail (re-render with updated values).
