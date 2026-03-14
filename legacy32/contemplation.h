/*
 * contemplation.h - AI Contemplation Period Interface
 *
 * Public interface for the mandatory 15-minute contemplation phase
 * during system startup.
 */

#ifndef CONTEMPLATION_H
#define CONTEMPLATION_H

#include <stdbool.h>

/**
 * Hold the AI process in contemplation state for 15 minutes
 * This function blocks until the contemplation period is complete
 */
void hold_ai_until_ready(void);

/**
 * Check if the neural registry is currently locked
 * @return true if locked, false if ready for AI operations
 */
bool is_neural_registry_locked(void);

/**
 * Release the neural registry lock
 * Called automatically at the end of contemplation period
 */
void release_neural_registry_lock(void);

/**
 * Verify that the AI has proper i386 architectural knowledge
 * @return true if verification passes, false otherwise
 */
bool verify_i386_knowledge(void);

/**
 * Initialize the contemplation subsystem
 * @return 0 on success, negative on error
 */
int init_contemplation_subsystem(void);

#endif /* CONTEMPLATION_H */
