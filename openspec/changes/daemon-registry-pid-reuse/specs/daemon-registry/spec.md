# Spec Delta

## Purpose

Which records in the per-user daemon registry count as running daemons, how a live pid is checked against its record, and what happens to records that are stale or malformed.

## ADDED Requirements

### Requirement: A record is live only when its process is alive (D-05)

Listing the registry SHALL check every record's pid. A record whose process is not alive SHALL be removed from the registry on that same call and SHALL NOT be listed. A listing SHALL return only live records.

#### Scenario: The process is gone

- **WHEN** a record names a pid that no process holds
- **THEN** the record file is removed and `codegraph daemon` does not list it

#### Scenario: The process is alive and corroborated

- **WHEN** a record names a pid whose process is alive and whose start time corroborates the record
- **THEN** the record is listed with its repository root

### Requirement: A live pid is corroborated by its start time (WR-02, T-04-07-02)

Where the operating system reports a process's start time, a record or lockfile naming a live pid SHALL be stale when that process started more than 5 seconds after the recorded start time. A recorded time at or after the process start SHALL corroborate. Where the start time is unavailable, liveness alone SHALL decide.

#### Scenario: The pid was recycled

- **WHEN** a record's pid is alive but the process started an hour after the recorded start time
- **THEN** the record is stale, is removed, and is not listed

#### Scenario: A record from an older release stamped the wall clock

- **WHEN** a record's pid is alive and the recorded start time is later than the process start time, by any amount
- **THEN** the record is live and is listed

#### Scenario: A lockfile from an older release on a live daemon

- **WHEN** `.codegraph/daemon.lock` names a live pid with a recorded start time later than the process start time
- **THEN** acquiring the lock fails because the lock is held by a live process, and the lockfile is not removed

#### Scenario: Start times are unavailable

- **WHEN** the platform cannot report a process start time
- **THEN** a live pid corroborates its record regardless of the recorded time

### Requirement: Process start times are read on Linux and macOS

codegraph SHALL read a live process's start time from the operating system on Linux and on macOS, and SHALL stamp that time into the records and lockfiles it writes for its own pid.

#### Scenario: A daemon on macOS corroborates itself

- **WHEN** a daemon started on macOS registers itself and the registry is listed while it runs
- **THEN** its record is listed

#### Scenario: A recycled pid on macOS

- **WHEN** on macOS a record's pid is alive but belongs to a process that started an hour after the recorded time
- **THEN** the record is removed and is not listed

### Requirement: Malformed registry files are removed

A file in the registry directory with the `.json` suffix whose name is not `<pid>.json`, whose content does not decode as a record, or whose `pid` field differs from its name SHALL be removed on a listing and SHALL NOT be listed. Files without the `.json` suffix SHALL be left untouched.

#### Scenario: A file named by something other than a pid

- **WHEN** the registry holds `d28d43d85835d89e.json`
- **THEN** a listing removes it and does not list it

#### Scenario: A record that does not decode

- **WHEN** the registry holds `4242.json` whose content is not a record
- **THEN** a listing removes it and does not list it

#### Scenario: A record whose pid does not match its name

- **WHEN** the registry holds `4242.json` whose `pid` field is 5151
- **THEN** a listing removes it and does not list it

#### Scenario: A foreign file

- **WHEN** the registry holds `notes.txt`
- **THEN** a listing leaves it in place
