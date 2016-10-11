# CDS: Continuous Delivery μservice

CDS is a pipeline based Continuous Delivery μservice written in Go.


## Overview

CDS is a μservice composed of 4 different Go binaries and an Angular UI application.
The first 2 binaries are the api and the cli.
The third is a worker binary in charge of running pipelines in queue.
The fourth binary is a worker hatchery. It fetches the workers need from the API and spawn them accordingly.

### API

To start CDS api, the only mandatory dependency is a PostreSQL database dsn and a path to the directory containing other CDS binaries. Ex:

```
$ ./api --db-host=127.0.0.1 --db-user=cds --db-password=XX --download-directory=$GOPATH/bin
```

To get the best out of CDS api though, one should use all compatible third parties to ensure maximum security and availability:
 - Openstack Swift for artifact storage
 - Vault for cipher and app keys
 - SMTP for mail notification
 - SSL for end-to-end crypted communication
 - Redis for caching
 - LDAP for user management

### Worker

Workers connect to API using a `worker key`, previously generated by a user, thus sharing all its permissions.
It will then fetch all permitted actions in the queue and check action requirements against its own capabilities, before starting the build.

### Hatchery

Hatchery is a binary dedicated to spawn and kill worker in accordance with build queue needs.

There is 3 modes for hatcheries:

 * Local (Start workers on a single host)
 * Local Docker (Start worker model instances on a single host)
 * Mesos (Start worker model instances on a mesos cluster)
 * Swarm (Start worker on a docker swarm cluster)
 * Openstack (Start hosts on an openstack cluster)

#### Local mode

Hatchery starts workers directly on on the same host.

#### Docker mode

Hatchery starts workers inside docker containers on the same host.

#### Marathon mode

Hatchery starts workers on a mesos cluster using Marathon API.

#### Openstack mode

Hatchery starts workers on Openstack servers using Openstack Nova.

### SDK

A Go SDK is available at <>. It provide helper functions for all API handlers, with embedded authentification mechanism.

### CLI

Build on top of the SDK. the CLI provides all features available via API and UI.

### UI


## Advanced API configuration

### Vault

It is possible to configure CDS to fetch secret cipher keys from Vault.

Keys are needed for:
 - AES+HMAC secret variable cipher key (looking for "cds/aes-key")
 - OAUTH2 Application secret for Stash and Github integration ("cds/repositoriesmanager-secrets-%s")


```
 --vault-host string                   Vault hostname (default "local-insecure")
 --vault-insecure-secrets-dir string   Load secrets from directory (default ".secrets")
 --vault-key string                    Vault application key (default "cds")
 --vault-password string               Vault password key
```

### Artifact Storage

 Artifacts are either stored on API filesystem or on Openstack Swift to garantee High Availabilty.

```
 --artifact-mode string                Artifact Mode: openstack or filesystem (default "filesystem")
```

### Caching

 Cache from database is enabled in process by default. To avoid high memory consumption, Redis caching is available.

```
 --cache string                        Cache : local|redis (default "local")
 --cache-ttl int                       Cache Time to Live (seconds) (default 600)
 --redis-host string                   Redis hostname (default "localhost:6379")
 --redis-password string               Redis password
```

### Notification

### SMTP

SMTP should be enabled to allow user account creation.

```
 --smtp-from string                    SMTP From
 --smtp-host string                    SMTP Host
 --smtp-password string                SMTP Password
 --smtp-port string                    SMTP Port
 --smtp-tls                            SMTP TLS
 --smtp-user string                    SMTP Username
```


### LDAP

Users can be fetched from an external LDAP. If activated, user creation directly in CDS is disabled.

```
 --ldap-base string                    LDAP Base
 --ldap-dn string                      LDAP Bind DN (default "uid=%s,ou=people,{{.ldap-base}}")
 --ldap-enable                         Enable LDAP Auth mode : true|false
 --ldap-host string                    LDAP Host
 --ldap-port int                       LDAP Post (default 636)
 --ldap-ssl                            LDAP SSL mode (default true)
 --ldap-user-fullname string           LDAP User fullname (default "{{.givenName}} {{.sn}}")
```

### Database

```
 --db-host string                      DB Host (default "localhost")
 --db-maxconn int                      DB Max connection (default 20)
 --db-name string                      DB Name (default "cds")
 --db-password string                  DB Password
 --db-port string                      DB Port (default "5432")
 --db-sslmode string                   DB SSL Mode: require (default), verify-full, or disable (default "require")
 --db-timeout int                      Statement timeout value (default 3000)
 --db-user string                      DB User (default "cds")
```

### Logging

API logs can either be printed on stdout or send in a dedicated table in database

```
 --db-logging                          Logging in Database: true of false
```

## Database configuration

4 files are available in sql/ folder, containing tables and constraints declarations.

### PostgreSQL

```
psql -U postgres -d postgres -h <dbHost> -p <dbPort> -a -f sql/func.sql
psql -U postgres -d postgres -h <dbHost> -p <dbPort> -a -f sql/create_table.sql
psql -U postgres -d postgres -h <dbHost> -p <dbPort> -a -f sql/create_index.sql
psql -U postgres -d postgres -h <dbHost> -p <dbPort> -a -f sql/create_foreign-key.sql
```
## Links

- *OVH home (us)*: https://www.ovh.com/us/
- *OVH home (fr)*: https://www.ovh.com/fr/


## License

3-clause BSD