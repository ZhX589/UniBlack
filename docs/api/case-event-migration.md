# Case to Event API Migration

The Event API is the canonical public read model. Legacy Case APIs remain
available only during the compatibility window ending on Thu, 31 Dec 2026
23:59:59 GMT.

| Legacy endpoint | Replacement | Compatibility behavior |
| --- | --- | --- |
| `GET /api/v1/cases/{id}` | `GET /api/v1/events/{id}` | The Case response retains `description` for existing clients. Event responses use `details`. |
| `GET /api/v1/subjects/{id}/cases` | `GET /api/v1/subjects/{id}` and linked Event reads | The legacy list retains its `cases` wrapper. New clients use the Subject and Event resources. |
| `GET /api/cases/{id}` | `GET /api/events/{id}` | Existing Case management routes remain available only for historical records. Other Case collection, mutation, history, evidence, and appeal routes have no direct Event endpoint replacement. |

Every legacy Case response includes these headers:

```text
Deprecation: true
Sunset: Thu, 31 Dec 2026 23:59:59 GMT
Link: </api/v1/events/{id}>; rel="successor-version"
Warning: 299 - "Case API is deprecated; migrate to Event API before sunset"
```

Clients should use `/api/v1/events/{id}` for Event reads and consume canonical
Event fields such as `details`, `occurred_from`, `occurred_to`, and `status`.
Do not add new integrations to Case routes. Compatibility-only fields remain on
legacy responses where existing callers require them: Subject `case_count`,
Case-list `cases`, and Case `description`.

The `Link` header appears only when a legacy route has a direct successor. All
other legacy Case routes still include `Deprecation`, `Sunset`, and `Warning`
headers, but omit `Link` rather than pointing to an unresolved route template.

`MINIO_PUBLIC_BASE` is only appropriate when a reverse proxy or the object
store itself intentionally exposes the configured object prefix to clients.
UniBlack does not configure a public bucket policy; operators must provide the
proxy or equivalent controlled public-read mechanism when returning these URLs.
