# Unprotected ID-mapper

The unprocted server is intended to run inside a trusted environment with a audit logging layer prior to access. All endpoints (including bulk endpoints) are available to whoever can connect to the server.

# (un)RESTfulness

The server is build to be simple to use and comporises on several REST best practices. The most major one is the at GET-request will mutate data. A Get call to either `/q/` or `/b/` with return existing records or create one which is returned. As such there is no distinction for the user if the recorded existed or not prior to the query (timing attacks excluded). Another mutation happens for every query containing a `&ttl=10m` (or other duration value) as it will update all mathced records with the new TTL.

# Clearing keys

It is only possible to clear by full paths. As such if you have the following keys

`Aa/Bb/Cc`
`Aa/Bb/Dd`
`Aa/Bb/Ee`

It is fully possible to clear the keys `Aa`, `Aa/Bb` but not `A` or `Aa/B` as they are interepted as `A/` and `Aa/B/`
