# spec-05-ui-request-response-edit

## Goal

Move response editing from the channel sidebar to the request detail panel, so users
can configure the response for each path independently by selecting a specific captured
request and editing its rule.

## Context

`ui/static/index.html` currently shows a status code badge next to each channel in the
sidebar and exposes an "Edit response" item in the channel context menu. Both must be
removed. The right home for response editing is the request detail panel: a user selects
a captured request, sees the response that will be returned for that method+path, and
can update it inline.

After spec-03, `GET /_/api/channels/{slug}/requests` and the single-request GET both
include `response_status` (int) and `response_body` (raw JSON) on each request object.
`PATCH /_/api/channels/{slug}/requests/{id}` accepts `{ "response_status", "response_body" }`
and returns the updated request or a JSON error body.

Two requests with the same method+path share a rule: editing one updates the rule for
all of them. The UI does not need to communicate this — it is an API-level behaviour.

## Out of scope

- Changes to any Go source files (specs 01–04).
- Pretty-printing or syntax highlighting in the response body textarea.
- Client-side validation of the JSON body (the server validates; surfacing inline errors
  from the 400 response is sufficient).

## Steps

### Revert (remove channel-level UI from the previous implementation)

1. Remove the `.status-badge` CSS rule.
2. Remove the `.modal-textarea` and `.modal-label` CSS rules.
3. Remove the Edit Response modal HTML block (`id="edit-response-backdrop"`).
4. Remove the `ctx-edit-response` item from the context menu.
5. In `renderChannels()`, remove the `badge` element creation and its `appendChild` call.
6. Remove the DOM refs `editResponseBackdrop`, `editStatusInput`, `editBodyInput`,
   `editResponseError`.
7. Remove `openEditResponseModal`, `closeEditResponseModal`, the `ctx-edit-response`
   click handler, the `btn-edit-response-cancel` click handler, the backdrop click
   handler on `editResponseBackdrop`, and the `btn-edit-response-save` click handler.
8. In the `keydown` Escape handler, remove the `closeEditResponseModal()` call.

### Add (request-level response editing in the detail panel)

9. **CSS** — Add a `.response-section` label style consistent with `.section-label`:
   ```css
   /* reuses .section-label — no new class needed */
   ```
   Add styles for the Edit Response modal that will live in the request detail:
   ```css
   .modal-textarea {
     width: 100%;
     background: var(--bg-base);
     border: 1px solid var(--border-hi);
     border-radius: 4px;
     padding: 7px 10px;
     font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
     font-size: 12px;
     color: var(--text-pri);
     outline: none;
     resize: vertical;
     min-height: 80px;
     margin-bottom: 6px;
   }
   .modal-textarea:focus { border-color: var(--accent); }
   .modal-textarea::placeholder { color: var(--text-muted); }
   .modal-label {
     display: block;
     font-size: 11px;
     color: var(--text-muted);
     margin-bottom: 4px;
   }
   ```

10. **HTML** — Add an Edit Response modal (separate from the New Channel modal):
    ```html
    <!-- ── Edit Response Modal ──────────────── -->
    <div class="modal-backdrop hidden" id="edit-response-backdrop"
         role="dialog" aria-modal="true" aria-labelledby="edit-response-title">
      <div class="modal">
        <div class="modal-title" id="edit-response-title">Edit Response</div>
        <div class="modal-sub">Sets the response for all requests to this method + path.</div>
        <label class="modal-label" for="edit-status-input">Status code</label>
        <input class="modal-input" id="edit-status-input" type="number"
               min="100" max="599" placeholder="200" autocomplete="off">
        <label class="modal-label" for="edit-body-input">Response body (JSON)</label>
        <textarea class="modal-textarea" id="edit-body-input"
                  placeholder="{}" spellcheck="false"></textarea>
        <div class="modal-error" id="edit-response-error"></div>
        <div class="modal-actions">
          <button class="btn btn-ghost" id="btn-edit-response-cancel">Cancel</button>
          <button class="btn btn-primary" id="btn-edit-response-save">Save</button>
        </div>
      </div>
    </div>
    ```

11. **JS — DOM refs** — Add:
    ```js
    const editResponseBackdrop = $('edit-response-backdrop');
    const editStatusInput      = $('edit-status-input');
    const editBodyInput        = $('edit-body-input');
    const editResponseError    = $('edit-response-error');
    ```

12. **JS — `renderDetail(req)`** — Add a "Response" section after the Body section:
    ```html
    <div class="section-label" style="margin-top:24px">Response</div>
    <div class="overview-grid">
      <div class="overview-row">
        <div class="overview-key">Status</div>
        <div class="overview-val">${escHtml(String(req.response_status))}</div>
      </div>
      <div class="overview-row">
        <div class="overview-key">Body</div>
        <div class="overview-val">${escHtml(JSON.stringify(req.response_body))}</div>
      </div>
    </div>
    <div style="margin-top:10px">
      <button class="btn btn-ghost" id="btn-edit-response">Edit response</button>
    </div>
    ```
    After the `innerHTML` assignment, wire up the button:
    ```js
    const btnEditResponse = detailArea.querySelector('#btn-edit-response');
    if (btnEditResponse) {
      btnEditResponse.addEventListener('click', () => openEditResponseModal(req));
    }
    ```

13. **JS — `openEditResponseModal(req)` and `closeEditResponseModal()`**:
    ```js
    function openEditResponseModal(req) {
      editStatusInput.value = req.response_status;
      editBodyInput.value = JSON.stringify(req.response_body);
      editResponseError.textContent = '';
      editResponseBackdrop.dataset.slug = req.channel_slug;
      editResponseBackdrop.dataset.id   = req.id;
      editResponseBackdrop.classList.remove('hidden');
      editStatusInput.focus();
    }

    function closeEditResponseModal() {
      editResponseBackdrop.classList.add('hidden');
      editResponseBackdrop.dataset.slug = '';
      editResponseBackdrop.dataset.id   = '';
    }
    ```

14. **JS — Wire up cancel, backdrop click, Escape, and save**:
    ```js
    $('btn-edit-response-cancel').addEventListener('click', closeEditResponseModal);

    editResponseBackdrop.addEventListener('click', e => {
      if (e.target === editResponseBackdrop) closeEditResponseModal();
    });
    ```
    Extend the existing `keydown` Escape handler to also call `closeEditResponseModal()`.

    Save handler:
    ```js
    $('btn-edit-response-save').addEventListener('click', async () => {
      const slug = editResponseBackdrop.dataset.slug;
      const id   = editResponseBackdrop.dataset.id;
      if (!slug || !id) return;

      const statusVal = parseInt(editStatusInput.value, 10);
      let bodyVal;
      try {
        bodyVal = JSON.parse(editBodyInput.value);
      } catch {
        editResponseError.textContent = 'Response body must be valid JSON.';
        return;
      }

      editResponseError.textContent = '';

      try {
        const res = await fetch(
          `/_/api/channels/${encodeURIComponent(slug)}/requests/${encodeURIComponent(id)}`,
          {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ response_status: statusVal, response_body: bodyVal }),
          }
        );

        if (res.ok) {
          const updated = await res.json();
          closeEditResponseModal();
          // Re-render the detail panel with the updated request data.
          renderDetail(updated);
        } else {
          let msg = `Error ${res.status}`;
          try {
            const data = await res.json();
            if (data.error) msg = data.error;
          } catch {}
          editResponseError.textContent = msg;
        }
      } catch (e) {
        editResponseError.textContent = `Network error: ${e.message}`;
      }
    });
    ```

## Acceptance criteria

- No status badge appears in the channel sidebar.
- The channel context menu has no "Edit response" item.
- Selecting a captured request shows a "Response" section in the detail panel with
  the current `response_status` and `response_body`.
- Clicking "Edit response" opens the modal pre-filled with the request's current values.
- Submitting valid values (e.g., status 418, body `{"teapot":true}`) issues a PATCH;
  on success the modal closes and the detail panel re-renders showing the new values.
- Submitting an invalid status or body shows the server error inline without closing
  the modal.
- Cancelling the modal makes no API call.
- Two requests with the same method+path: editing one causes the detail panel of the
  other (when selected) to show the same response_status/response_body on the next
  render (the value comes from the API, not local state).

## Dependencies

- spec-03 (response_status/response_body on requestJSON; PATCH requests endpoint).
