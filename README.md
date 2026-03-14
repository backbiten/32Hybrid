| `third_party/kali` | https://www.kali.org/ | security auditing and penetration testing tools |
| `third_party/ipfire` | https://www.ipfire.org/ | versatile and state-of-the-art Open Source firewall |

## Startup Contemplation Gate

Both the control plane and runner binaries now honor a mandatory 15-minute
“contemplation” period on startup to re-center on 32-bit (i386/486) limits.
During this window the process is held in a WAIT state, emits a
`Synchronizing Neural Root...` progress bar, and rotates i386 reminders
covering GDT/IDT hygiene, CR0/CR3 paging discipline, and avoidance of 64-bit
opcodes while it re-reads the Open386 toolchain notes. The service does not
accept work until the timer ends.

- Default duration: 900 seconds (15 minutes).
- Override for development/tests: set `HYPER32_CONTEMPLATION_SECONDS` to a
  positive integer (seconds) or Go duration string (e.g. `45` or `2m30s`).
