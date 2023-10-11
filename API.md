# Tenant Audit Log API

This document describes the HTTP/JSON API used to access the tenant audit log
service.

## Events

Events are captured when a user performs certain actions. The Tenant Audit Log
API allows access to see the events generated for all of a tenants users. This
can be useful to proactively notify users that a guess attempt has been made on
their secret.

The following events are captured

1. Registered: When a user registers a secret.
2. Guess Used: When a user make an attempt to recover the secret and consumes a guess.
3. Share Recovered: When a user provided the correct PIN and the secret share was recovered.
4. Deleted: When the user deleted their secret.

Each event will include:

- A timestamp of when it happened.
- A unique identifier of the event.
- The event type, one of registered, guess_used, share_recovered, deleted.
- An acknowledgement identifier used to indicate that the event has been consumed.
- A user identifier
- For guess_used, it will also include the number of guesses allowed and number of guesses used counts.

## Fetching Events

The `/tenant_log` endpoint accepts a HTTPS POST request which will return some
number of events. If there are no events left to be fetched, this request may
block for up to 20 seconds waiting to see if a new event is generated. The
request payload to the `/tenant_log` endpoint should be a json body
```json
{
    "ack": [],
    "page_size": 5
}
```

The `page_size` parameter indicates the maximum number of events to return
from this request. If not provided it defaults to 1. It is capped at a maximum
of 200. You may get less than the requested `page_size` number of results even
if that number of events exist.

The `ack` array can contain ack id's from previously fetched events. This allows
a consumer of the API to call this endpoint in a loop where the next request for
events acknowledges the previous batch of events.

The response from the `/tenant_log` endpoint is a JSON body of the form
```json
{
    "events":[{
        "id":"9321537408648904",
        "ack":"UTcZCGhRDk9eIz81IChFGwMIFAV8fUpbUzQNFCkaVwx2cn1hdWpbG1MDsXUZI",
        "when":"2023-10-11T20:17:02.342Z",
        "user_id":"c82486815b36aaac09fd2d56ca8fbaf1f4f0519625d1a7c6869e49e9f4c0e5a6",
        "event":"guess_used",
        "num_guesses":2,
        "guess_count":1
    }]
}
```
There can be between 0 and `page_size` number of events in the response.

## Event Acknowledgement

Events have at least once delivery semantics and must be explicitly acknowledged
by the consumer. Once returned from the `/tenant_log` endpoint the consumer has
10 seconds to acknowledge the event. If not acknowledged the event will be
returned by a subsequent `/tenant_log` request.

Events are returned in no particular order, and may not be in timestamp order.

In addition to the `/tenant_log` endpoint, the `/tenant_log/ack` endpoint can be
used to acknowledge events without fetching any new events. Requests to
`/tenant_log/ack` should be the same as the `/tenant_log` endpoint but without
the `page_size` field.

## Authentication

Access to the API endpoints requires a JWT token with the same properties as
required by the client SDK. In addition, the token should include a field
`scope` in its claims with the value `audit`.

The `tokens` tool in the client SDK can help generate and validate JWT tokens.

The token should be included in an `Authorization` header in the HTTP request.
The header value should be prefixed with `Bearer `. e.g. the resulting header
should be of the form `Authorization: Bearer <jwt_token>`


## User identifier

To prevent the possible storage of PII the user identifier is stored as a hash.
The hash is computed as `sha256(utf8(tenantName+":"+userId))` where `tenantName`
and `userId` were extracted from the `Issuer` and `Subject` fields of the JWT
token used. In order to map these events back to users, the same hash will need
to be computed and associated with the tenants user data.

For example given an issuer of `test` and a subject of `121314` the resulting
user_id hash is `447ddec5f08c757d40e7acb9f1bc10ed44a960683bb991f5e4ed17498f786ff8`

## Expiry

Unacknowledged events are available for at least 7 days. Events older than that
may be expired and deleted without further notice.
