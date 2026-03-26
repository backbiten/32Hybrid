/*
 * legacy32/contemplation.c — Neural Registry lock for the 32-bit AI Teacher.
 *
 * This module implements the kernel-side "Contemplation Period" described in
 * the 32Hybrid spec.  It holds the AI Teacher process in a WAIT state for
 * 15 minutes (900 seconds) while it verifies its understanding of core i386
 * architectural concepts.  Progress is emitted to stdout so that the
 * WinStratch "Synchronizing Neural Root..." dialog can render the progress bar
 * with per-second concept descriptions.
 *
 * Kernel lock contract:
 *   Call hold_ai_until_ready() at process startup.  The function blocks until
 *   the Neural Registry lock is released, after which the AI Teacher may
 *   access the Micro-Bus and issue tasks to the IA Student.
 *
 * Build for 32-bit target (Open386 / GCC):
 *   gcc -m32 -march=i386 -O2 -o contemplation contemplation.c
 */

#include <stdio.h>
#include <unistd.h>

#define CONTEMPLATION_SECONDS 900

/* concept_entry — maps a [start, end) second range to an i386 topic. */
struct concept_entry {
    int         start_sec;
    int         end_sec;
    const char *subject;
};

static const struct concept_entry CONCEPTS[] = {
    {   0, 180, "GDT (Global Descriptor Table)"             },
    { 180, 360, "IDT (Interrupt Descriptor Table)"          },
    { 360, 540, "Memory Segments and Paging (CR3/CR0)"      },
    { 540, 720, "32-bit Instruction Set Limits"             },
    { 720, 900, "Open386 Toolchain Curriculum Review"       },
};

#define CONCEPT_COUNT (sizeof(CONCEPTS) / sizeof(CONCEPTS[0]))

/* concept_for_second — returns the i386 topic string for a given second. */
static const char *concept_for_second(int sec)
{
    size_t i;
    for (i = 0; i < CONCEPT_COUNT; i++) {
        if (sec >= CONCEPTS[i].start_sec && sec < CONCEPTS[i].end_sec)
            return CONCEPTS[i].subject;
    }
    return "Finalising Neural Root synchronisation";
}

/*
 * hold_ai_until_ready — block the AI Teacher for the full contemplation
 * period, emitting WinStratch-compatible progress lines to stdout.
 *
 * After this function returns the caller may release the Neural Registry lock
 * and allow the AI Teacher to access the Micro-Bus.
 */
void hold_ai_until_ready(void)
{
    int sec;

    printf("[Neural Root]   0%% (  0/%d) Synchronizing Neural Root"
           " — Contemplation Period begins.\n",
           CONTEMPLATION_SECONDS);
    fflush(stdout);

    for (sec = 1; sec <= CONTEMPLATION_SECONDS; sec++) {
        sleep(1);
        printf("[Neural Root] %3d%% (%3d/%d) Contemplating i386: %s\n",
               (sec * 100) / CONTEMPLATION_SECONDS,
               sec,
               CONTEMPLATION_SECONDS,
               concept_for_second(sec - 1));
        fflush(stdout);
    }

    printf("[Neural Root] 100%% (%d/%d) Neural Registry lock released."
           " AI Teacher may now access the Micro-Bus.\n",
           CONTEMPLATION_SECONDS, CONTEMPLATION_SECONDS);
    fflush(stdout);
}
