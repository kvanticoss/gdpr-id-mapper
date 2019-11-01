# gdpr-id-mapper

[![Go Report Card](https://goreportcard.com/badge/github.com/kvanticoss/gdpr-id-mapper?style=flat-square)](https://goreportcard.com/report/github.com/kvanticoss/gdpr-id-mapper)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/kvanticoss/gdpr-id-mapper)

Translation-layer from internal (user) ids to ephemeral public ids which can be used in communicating with markting and analytics platforms.

# State
Fully functional, but unprotected. Would work fine in a trusted environment where brute force attacks can be blocked ahead of query lookups. More feature rich versions are planned.

# Building
`make build-all`

# Running
Run an unprotected development server on port 5000 with random global salt.
`make run`

# Deploying

Update the variable `REGISTRY` in the Makefile and run `make image` to build an docker image (will not be pushed). The image is ready for deployment `docker run ....`. To gain persistence `-default-salt=...` and `-db=/tmp/dbpath` should both be set and the db-path should point to a a non ephemeral path.

# Purpose
NOTE THIS IS **NOT** LEGAL ADVICE; implied or otherwise. This repository provides a tool (without any warranty), not a solution.

User IDS (be they emails, user-ids, order-ids, IPs etc) can all be interpreted to be personally identifiable information according to GDPR. As such, care should be taken before these IDs are shared with external partners (this repository does not offer any legal advice on the matter). One issue with sharing data externally is the increased complexity of deleting shared data. It can therefore be easier to share ids which can be de-associated from an individual "Ephemeral IDs" when needed since GDPR only deals with personally identifiable information (once again seek legal advice to understand what you can and can not do with personal data).

The idea is simple. Instead of sharing your internal customer-id, share an random representation of it; a "token-id".

**Internal system**

| Internal Customer Id | Ephemeral ID                                | data                     |
| -------------------- | ------------------------------------------- | ------------------------ |
| foo@bar.com          | AqxX1v2GWhhcQYD9L5b5RuR3UVj9SL0UoKc1fU5rHj8 | viewed page: programming |
| hello@world.com      | _Bs4N4gBp_xXbHO7wqDGthDvtOvPtoYpAd7aYbAGqDo | viewed page: coding      |

Where `Ephemeral ID` and `data` is sent to an external partner. If we later need to anonymize data tied to `foo@bar.com`, we can take the necessary actions on our internal systems by simply re-generating (or remove) the ephemeral id for `foo@bar.com`.

**After anonymization: Internal system**

| Internal Customer Id | Ephemeral ID                                | data                |
| -------------------- | ------------------------------------------- | ------------------- |
| hello@world.com      | _Bs4N4gBp_xXbHO7wqDGthDvtOvPtoYpAd7aYbAGqDo | viewed page: coding |

**After anonymization: External system**

| Ephemeral ID                                | data                     |
| ------------------------------------------- | ------------------------ |
| AqxX1v2GWhhcQYD9L5b5RuR3UVj9SL0UoKc1fU5rHj8 | viewed page: programming |
| _Bs4N4gBp_xXbHO7wqDGthDvtOvPtoYpAd7aYbAGqDo | viewed page: coding      |

Since there is no longer any connection between `AqxX1v2GWhhcQYD9L5b5RuR3UVj9SL0UoKc1fU5rHj8` and `foo@bar.com` (anywhere) the first row can be considered anonymized and no no longer falls under GDPR.

The same results would be achieved if we instead replaced the ephemeral id with a new one.

**Internal system**

| Internal Customer Id | Ephemeral ID                                    | data                     |
| -------------------- | ----------------------------------------------- | ------------------------ |
| foo@bar.com          | **b9tHm5htMp4SMzU3whsEEShLf1pyRBYRdDFUytUnS2w** | viewed page: programming |
| hello@world.com      | _Bs4N4gBp_xXbHO7wqDGthDvtOvPtoYpAd7aYbAGqDo     | viewed page: coding      |

Since there is still no connection from the data to `foo@bar.com`

**After anonymization: External system**

| Ephemeral ID                                | data                     |
| ------------------------------------------- | ------------------------ |
| AqxX1v2GWhhcQYD9L5b5RuR3UVj9SL0UoKc1fU5rHj8 | viewed page: programming |
| _Bs4N4gBp_xXbHO7wqDGthDvtOvPtoYpAd7aYbAGqDo | viewed page: coding      |


**PLEASE NOTE**: This approach only works to anonymize data if there is NO other connection between the internal ID (`foo@bar.com`) and the external data(`viewed page: programming`). If there is any way to tie to data-point to the user (`foo@bar.com`) it is arguable still personal data and falls under GDPR. As such there are several other factors to consider and until the anonymization happens the external partner should be considered to hold personal data (and as such need a DPA)

# Usage


## Note on (un)RESTfulness

The server is build to be simple to use and compromises on several REST-best-practices. The most major one is the at GET-request will mutate data. A Get call to either `/q/` or `/b/` with return existing records or create one which is returned. As such there is no distinction for the user if the recorded existed or not prior to the query (timing attacks excluded). Another mutation happens for every query containing a `&ttl=10m` (or other duration value) as it will update all mathced records with the new TTL. A call to `/c/` will clear data under that prefix.

## Querying

Query the server with the ID you need to translate to a random public representation.

`curl localhost:5000/q/foo@bar.com`

You will be returned with a record containing a public token for the internal id (`PublicId`)
```
{"Status":"ok","Msg":"","Payload":{"OriginalID":"foo@bar.com","PublicID":"P5K9Fkix0Dj7xUzTdmoFKnCksBEvGi538KufDL3cDnE=" "AliveUntil":"2020-10-29T16:58:00.8954442+01:00"}}
```

Consecutive queries will yield the same PublicId up until the `AliveUntil` timestamp has been passed, at which point a new Id will be returned.

if you instead want to refresh the `AliveUntil` timestamp; simply provide a ttl parameter in the request `&ttl=80000h` to set the AliveUntil 80000 hours into the future

`curl localhost:5000/q/foo@bar.com?ttl=80000h`
```
{"Status":"ok","Msg":"","Payload":{"OriginalID":"foo@bar.com","PublicID":"P5K9Fkix0Dj7xUzTdmoFKnCksBEvGi538KufDL3cDnE=","AliveUntil":"2028-12-15T01:02:13.9584236+01:00"}}
```

Setting a negative ttl will invalidate the record on the next request.


The system also understands hierarchical IDs.

`curl localhost:5000/q/foo@bar.com/order/orderId1235`
```
{"Status":"ok","Msg":"","Payload":{"OriginalID":"foo@bar.com/order/orderId1235","PublicID":"ULmFC1QdsNLsXxf6kt3c9Gu47RHC9vQEkRh2ZEqsGA8=","AliveUntil":"2020-10-29T17:05:56.7721761+01:00"}}
```

Structuring keys in a hierarchy makes it easier to delete (clear) ephemeral ids belong to one user (see Clearing Keys) based on a prefix. It also allow you to create unique publicIDs for each partner.


`curl localhost:5000/q/foo@bar.com/order/orderId1235/analyticsParter1`
```
{"Status":"ok","Msg":"","Payload":{"OriginalID":"foo@bar.com/order/orderId1235/analyticsParter1","PublicID":"-LEBA8qRui0_wgfZyIA1fYBWz8qAq0oekT6sYIABOK4=","AliveUntil":"2020-10-29T17:08:06.3893041+01:00"}}
```

`curl localhost:5000/q/foo@bar.com/order/orderId1235/analyticsParter2`
```
{"Status":"ok","Msg":"","Payload":{"OriginalID":"foo@bar.com/order/orderId1235/analyticsParter2","PublicID":"FQHfYbaRf2vZTGnTTUEhMp_saPvMWLEeHMS9oP1lbzg=","AliveUntil":"2020-10-29T17:08:50.5471761+01:00"}}
```


## Clearing keys

To clear a prefix of keys replace `/q/` with `/c/`

`curl localhost:5000/c/foo@bar.com/order`
```
{"Status":"ok","Msg":"Removed all(4) records starting with foo@bar.com/order","Payload":null}
```

It is possible to clear all keys under a prefix; but only by a full path.

If the given IDs have historically been inserted:
`Aa/Bb/Cc`
`Aa/Bb/Dd`
`Aa/Bb/Ee`
It is fully possible to clear the keys `Aa`, `Aa/Bb` but not `A` or `Aa/B` as they are interpreted as `A/` and `Aa/B/`


# Architecture

The system is structured with a zero PII at rest principle. The original IDs that are sent to the server are only keep in memory for the duration of the request (excluding non GC RAM). Any data that is stored in the DB is hashed one or multiple times; each with an individual salt of at least 32 bytes.

When the server is started it should be provided with a "global-salt" which will be included with all hashes; as such, if the global salt is forgotten ALL records in the system can be considered anonymized (starting the server with 2 different global salt but same database would mean queries on the same Internal IDs would yield different results). You can consider the global-salt to be the  equivalent of an encryption key (but it is not)

Even with the global-salt; entires in the db are not recognizable to any external observer without the original ID to query the db against. However; a leaked db would be prone to brute force attacks and should thus be keep safe with (rate) limited query access.
