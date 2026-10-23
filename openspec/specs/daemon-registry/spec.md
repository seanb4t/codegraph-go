# daemon-registry Specification

## Purpose
Which records in the per-user daemon registry count as running daemons, how a live pid is checked against its record, and what happens to records that are stale or malformed.

## Requirements

### Requirement: A record is live only when its process is alive (D-05)

Listing the registry SHALL check every record's pid. A record whose process is not alive SHALL be removed from the registry on that same call and SHALL NOT be listed. A listing SHALL return only live records. A pid outside 1 through 2147483647 SHALL NOT be treated as alive and SHALL NOT be signaled.

#### Scenario: The process is gone

- **WHEN** a record names a pid that no process holds
- **THEN** the record file is removed and `codegraph daemon` does not list it

#### Scenario: A pid outside the range of a process id

- **WHEN** the registry holds `4294967295.json` whose `pid` field is 4294967295
- **THEN** a listing removes it, does not list it, and `codegraph daemon stop --all` sends it no signal

#### Scenario: The process is alive and corroborated

- **WHEN** a record names a pid whose process is alive and whose start time corroborates the record
- **THEN** the record is listed with its repository root

### Requirement: A live pid is corroborated by its start time (WR-02, T-04-07-02)

Where the operating system reports a process's start time, a record or lockfile naming a live pid SHALL be stale when that process started more than 5 seconds after the recorded start time. A recorded time at or after the process start SHALL corroborate. Where the start time cannot be read, liveness alone SHALL decide, and it SHALL be checked again after the failed read.

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

#### Scenario: The process exits while its start time is read

- **WHEN** a record names a pid whose process is alive at the liveness check and exits before its start time is read
- **THEN** the record is stale, is removed, and is not listed

### Requirement: Process start times are read on Linux and macOS

codegraph SHALL read a live process's start time from the operating system on Linux and on macOS, and SHALL stamp that time into the records and lockfiles it writes for its own pid.

#### Scenario: A daemon on macOS corroborates itself

- **WHEN** a daemon started on macOS registers itself and the registry is listed while it runs
- **THEN** its record is listed

#### Scenario: A recycled pid on macOS

- **WHEN** on macOS a record's pid is alive but belongs to a process that started an hour after the recorded time
- **THEN** the record is removed and is not listed

### Requirement: Malformed registry files are removed

A file in the registry directory with the `.json` suffix whose name is not `<pid>.json` SHALL be removed on a listing. A `<pid>.json` whose content does not decode as a record, or whose `pid` field differs from its name, SHALL NOT be listed; it SHALL be removed on a listing when no process holds that pid and left in place while one does. Files without the `.json` suffix SHALL be left untouched.

#### Scenario: A file named by something other than a pid

- **WHEN** the registry holds `d28d43d85835d89e.json`
- **THEN** a listing removes it and does not list it

#### Scenario: A record that does not decode

- **WHEN** the registry holds `4242.json` whose content is not a record, and no process holds pid 4242
- **THEN** a listing removes it and does not list it

#### Scenario: A record whose pid does not match its name

- **WHEN** the registry holds `4242.json` whose `pid` field is 5151, and no process holds pid 4242
- **THEN** a listing removes it and does not list it

#### Scenario: A malformed record under a live pid

- **WHEN** the registry holds `<pid>.json` whose content is not a record, and a process holds that pid
- **THEN** a listing does not list it and leaves the file in place

#### Scenario: A foreign file

- **WHEN** the registry holds `notes.txt`
- **THEN** a listing leaves it in place

### Requirement: An unreadable registry record is handled like a malformed one

A `<pid>.json` that a listing cannot read, for any reason other than the file no longer existing, SHALL NOT be listed. It SHALL be removed on that listing when no process holds that pid, and left in place while one does.

#### Scenario: An unreadable record whose pid is dead

- **WHEN** the registry holds `4242.json`, reading it fails with an error other than "does not exist", and no process holds pid 4242
- **THEN** a listing removes it and does not list it

#### Scenario: An unreadable record under a live pid

- **WHEN** the registry holds `<pid>.json`, reading it fails with an error other than "does not exist", and a process holds that pid
- **THEN** a listing does not list it and leaves the file in place

### Requirement: A registry file that cannot be removed is logged

When a listing fails to remove a registry file, codegraph SHALL write one line to stderr that names the file and the error. A removal that fails because the file no longer exists SHALL NOT be logged. One process SHALL write the line for a given file at most once.

#### Scenario: A stale record cannot be removed

- **WHEN** the registry holds a record whose pid no process holds, and removing the file fails with a permission error
- **THEN** the listing does not list it, and stderr holds one line naming the file and the error

#### Scenario: The same file fails again in the same process

- **WHEN** one process lists the registry twice and the same file cannot be removed either time
- **THEN** stderr holds that file's line once

### Requirement: Skipped registry records are reported

A skipped record is a `<pid>.json` that a listing leaves in place and does not list because a process holds its pid. Its file does not decode, names another pid, or cannot be read. A file left in place because its removal failed is not a skipped record. When records are skipped, `codegraph daemon` and `codegraph daemon stop` SHALL print one stderr warning line naming each one. Stdout and exit status SHALL not change.

#### Scenario: Stop all with a malformed record under a live pid

- **WHEN** the registry holds only a `<pid>.json` that does not decode, a process holds that pid, and the user runs `codegraph daemon stop --all`
- **THEN** stdout is `no running daemons`, stderr holds one warning line naming the file, and the command exits 0

#### Scenario: Listing with a skipped record

- **WHEN** the registry holds one live daemon's record and one skipped `<pid>.json`, and the user runs `codegraph daemon` off a terminal
- **THEN** stdout lists the live daemon exactly as it would without the skipped file, and stderr holds one warning line naming the skipped file

#### Scenario: The picker's stop action skips a record

- **WHEN** the user runs `codegraph daemon` on a terminal and chooses to stop all, and the stop's listing skips a record that the first listing did not
- **THEN** after the picker exits, stderr holds one warning line that names each record either listing skipped, each one once

#### Scenario: A stale record that cannot be removed is not a skipped record

- **WHEN** the only file a listing leaves in place is a stale record whose removal failed
- **THEN** `codegraph daemon` and `codegraph daemon stop` print no skipped-record warning

#### Scenario: Nothing was skipped

- **WHEN** a listing skips no record
- **THEN** `codegraph daemon` and `codegraph daemon stop` write no warning to stderr

### Requirement: A failed start-time read on macOS is logged

On macOS, a start-time read can fail for a reason other than no process holding the pid. Then codegraph SHALL write one line to stderr that names the error. It SHALL write that line at most once per process. Liveness then decides, as it does when start times are unavailable.

#### Scenario: The start-time read is denied

- **WHEN** on macOS the kernel denies every start-time read, the registry holds a valid record whose pid is live, and one process lists the registry twice
- **THEN** both listings list the record, and stderr holds one line naming the error

#### Scenario: No process holds the pid

- **WHEN** on macOS a start-time read fails because no process holds the pid
- **THEN** nothing is written to stderr
