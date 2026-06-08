# Mission Memory

Every completed mission creates a mission record. Stored in `docs/missions/`.

## Format

```md
# Mission Record: <title>

MISSION:
OBJECTIVE:
DIFFICULTY: TINY / SMALL / MEDIUM / LARGE / CRITICAL
MISSION_TYPE:
AGENTS_USED:
STATES_ENTERED:
FILES_CHANGED:
FILES_INSPECTED:
COMMANDS_RUN:
TESTS_RUN:
DECISIONS:
RISKS:
ASSUMPTIONS:
HUMAN_APPROVALS:
FINAL_OUTCOME:
FOLLOW_UP:
LESSONS_LEARNED:
DOCTRINE_UPDATES_NEEDED:
DOCUMENTATION_NEEDED:
SECTOR_Z_MEMORY_NOTES:
```

## Purpose

Mission Memory gives future agents historical context.

- Sector Z uses this for archaeology.
- Numbuh 999 uses this for documentation.
- Numbuh 5 uses this for review patterns.
- Numbuh 0 uses this for architecture memory.
- Numbuh 86 uses this for decommissioning history.

## Rule

Every mission that touches code, architecture, security, or deployment should produce a record.

TINY/QUICK_TASK missions may skip if no meaningful learning occurred.
