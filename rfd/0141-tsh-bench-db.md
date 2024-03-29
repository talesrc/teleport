---
authors: Gabriel Corado (gabriel.oliveira@goteleport.com)
state: draft
---

# RFD 0141 - `tsh bench` subcommands for databases

## Required Approvals

* Engineering: @smallinsky && @r0mant
* Product: @xinding33 || @klizhentas
* Security: @reedloden

## What
Introduce new subcommands, for benchmarking database access, which will work
similarly to `tsh bench` for SSH and Kubernetes benchmarking.

This RFD initial focus is on covering PostgreSQL and MySQL protocols.

## Why
The team is working towards understanding performance issues with database
access. This built-in tool will be used to generate database access load and
produce some client-side metrics that will be used to understand overall
performance better.

In addition, the command will also have a mode of direct running the load to the
databases (without Teleport proxying them). The team can use this to compare
Teleport performance with direct database access.

## Prior art

### `tsh bench` subcommands
The `tsh` already has commands for performing benchmarks. Those rely on the
structure defined at `lib/benchmark/benchmark.go`. It defines a small framework
to define benchmarks. By following it, we can rely on the tools already used on
the SSH and Kubernetes benchmarks, such as reports.

To help readers better understand how the database benchmark will be organized,
here is a summary of it:
* Benchmark suites: Implements a function that generates workload functions.
  The suite is called before the benchmark starts and returns a workload function.
  It has access to the Teleport Client.
* Workload function: Functions that will be repeatedly executed to grab data.
  Since the suite generates it, it can use information from the suite function.
  For example, on the SSH benchmark, the servers list used to select the server
  where the command will be executed is fetched on the suite function, and the
  workload function only connects to it.

#### Common options
The benchmark structure already supports common flags to parametrize the test
execution:
- `--rate`: Control how many workload functions are executed per second. For
  example, it can control how many connections will be sent to the target
  database per second.
- `--duration`: Defines for how long the test will be executed.

#### Results
The benchmark commands follow a standardized result output. There is also an
option to export the test results to a more detailed version using the `--export`
flag.

```code
$ tsh bench ...
* Requests originated: 9
* Requests failed: 0

Histogram

Percentile Response Duration
---------- -----------------
25         123 ms
50         125 ms
75         129 ms
90         130 ms
95         130 ms
99         130 ms
100        130 ms
```

### [`mysqlslap`](https://dev.mysql.com/doc/refman/5.7/en/mysqlslap.html)

> `mysqlslap` is a diagnostic program designed to emulate client load for a
MySQL server and to report the timing of each stage. It works as if multiple
clients are accessing the server.

It has to working formats:
* Auto-generates database tables and queries. When used like this, it has flags
  to customize table columns (set how many columns there will be) and choose the
  type of queries it will execute during the benchmark (for example, `read` will
  only execute SELECT queries).
* Custom queries. It has a parameter to setup a table and insert necessary data.
  And executes the custom query provided by the user.

## Details
The database subcommands will be divided per protocol. This gives more
flexibility for each protocol to require their arguments and flags.

The "root" command for each protocol will execute the following:
* Starts a connection with the database (using local proxy);
* Issue a ping-like query to ensure the connection is working. This also covers
  drivers that connect only when queries are executed. For the initial databases,
  we'll have the following executed:
    * PostgreSQL: `SELECT 1;` query.
    * MySQL: `mysql_ping` command (using `client.Conn.Ping()` function).
* Closes the connection.

User certificate generation and local proxy start will not affect the final
results. The connections will be issued to a single local proxy, reducing the
resources used locally.

The metrics generated by those subcommands will be restricted to raw end-to-end
timings (without any breakdown). It must be used alongside backend metrics to
have full detailed information on the service behavior.

```mermaid
sequenceDiagram
  participant tsh
  participant Teleport
  participant DB as Database

  tsh->>Teleport: Generate Certificates (tsh db login)

  loop every execution
    tsh->>Teleport: Establish connection
    Teleport->>DB: Connect
    DB->>tsh: Connection
    tsh->>DB: Ping
  end
```

### Commands
Both PostgreSQL and MySQL will have the same command arguments and flags.

```code
tsh bench postgres [--db-user=] [--db-name=] [database]
```

```code
tsh bench mysql [--db-user=] [--db-name=] [database]
```

| Flag        | Description |
| ----------- | ----------- |
| `--db-user` | Database user used to connect to the target database. The user must have enough permissions on the database to execute all the benchmark queries. |
| `--db-name` | Database name where benchmark queries will be executed. Depending on the database protocol, this isn't required. |
| `database`  | Teleport target database name or the direct database URI. Available databases can be retrieved by running `tsh db ls`. When using direct connection, the benchmark will issue connections directly to this database, and no Teleport is involved in the testing. It must contain all the connection information, including authentication credentials. The contents from --db-user and --db-name will be ignored when provided. |

Note: The `--rate` common flag will indicate how many connections per second will be
established.

**Examples:**
```bash
# Executes 25 connections per second during 10 seconds without errors but also
# without meeting the target requests.
$ tsh bench postgres --rate=25 --duration=10s --db-user=postgres --db-name=postgres postgres-dev

* Requests originated: 88
* Requests failed: 0

Histogram

Percentile Response Duration
---------- -----------------
25         1346 ms
50         3535 ms
75         5259 ms
90         5847 ms
95         6663 ms
99         7271 ms
100        7311 ms

# Executes 100 connections per second during 30 seconds with errors (only the
# last error is shown).
$ tsh bench postgres --rate 100 --duration 30s --db-user=postgres --db-name=postgres postgres-dev

* Requests originated: 1586
* Requests failed: 30
* Last error: failed to connect to `host=localhost user=postgres database=`: dial error (dial tcp [::1]:5433: connect: connection refused)

Histogram

Percentile Response Duration
---------- -----------------
25         3773 ms
50         4639 ms
75         5263 ms
90         6243 ms
95         6775 ms
99         10567 ms
100        22783 ms

# Executes 10 connections per second during 30 seconds, exporting the latency
# profile to the current path.
$ tsh bench postgres --rate 10 --duration 30s --db-user=postgres --db-name=postgres --export postgres-dev

* Requests originated: 30
* Requests failed: 0

Histogram

Percentile Response Duration
---------- -----------------
25         1346 ms
50         3535 ms
75         5259 ms
90         5847 ms
95         6663 ms
99         7271 ms
100        7311 ms
latency profile saved: latency_profile_2023-01-01_00:00:00.txt

# Load results from latency profile.
$ cat latency_profile_2023-01-01_00:00:00.txt
 Value  Percentile      TotalCount      1/(1-Percentile)

     123.000     0.000000            1         1.00
     123.000     0.005000            1         1.01
     123.000     0.010000            1         1.01
     ...
     563.000     1.000000           99          inf
#[Mean    =      225.061, StdDeviation   =      109.519]
#[Max     =      563.000, Total count    =           99]
#[Buckets =            6, SubBuckets     =         2048]
```

### Direct database connections
Directly targeting databases (not through Teleport) can be used by developers
quickly to generate baseline measures that can be used to determine how much
overhead Teleport adds compared to raw database connections.

The direct database access will be provided on as the database (instead of a
Teleport database name).

By providing a database URI, the `tsh` will connect directly to the database
without using any Teleport certificate.

**Example:**
```code
$ tsh bench postgres "postgres://hello@localhost:5432"
```

## Security

### Bypass MFA or access checker
The command will use the same flow used by `tsh db login` to issue certificates,
including support to MFA. The main difference is that the issue credentials
won't be persisted on disk.

### Malicious URI on the direct database connection
Users may try to send malicious URI when connecting directly to databases.
Teleport doesn't validate the input provided; it is passed directly to the
database client responsible for validating it. Client execution is entirely
done locally. Nothing is sent to the Teleport server.

## Future work

### Run benchmark on multiple databases
As in the first versions, benchmarks are executed in the specified database.
Providing a predicate query enables users to select databases to execute
the benchmarks.

```code
$ tsh bench postgres --query 'labels["env"] == "prod"'
```

### Custom query execution
It will work similarly to the connection suite. The only difference is that
instead of executing a ping-like query, `tsh` will execute a user-provided query.
This allows benchmarking of more scenarios affected by what is performed on the
connection. For example, developers can benchmark how database access with large
queries, generating workload for packet parsing and audit logging (for databases
that support it).