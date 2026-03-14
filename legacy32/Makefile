# Makefile for Legacy32 Contemplation Period
#
# This builds the C implementation of the contemplation period mechanism

CC = gcc
CFLAGS = -Wall -Wextra -O2 -std=c99 -D_POSIX_C_SOURCE=200809L
LDFLAGS = -lm

TARGET = contemplation
OBJS = main.o contemplation.o

# Default target
all: $(TARGET)

# Build the main executable
$(TARGET): $(OBJS)
	$(CC) $(CFLAGS) -o $(TARGET) $(OBJS) $(LDFLAGS)
	@echo "Built $(TARGET) successfully"

# Compile object files
%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

# Dependencies
main.o: main.c contemplation.h
contemplation.o: contemplation.c contemplation.h

# Clean build artifacts
clean:
	rm -f $(TARGET) $(OBJS)
	rm -f /tmp/contemplation_progress /tmp/neural_registry_unlocked
	@echo "Cleaned build artifacts"

# Install to system (requires root)
install: $(TARGET)
	install -m 755 $(TARGET) /usr/local/bin/32hybrid-contemplation
	@echo "Installed to /usr/local/bin/32hybrid-contemplation"

# Run the contemplation period
run: $(TARGET)
	./$(TARGET)

# Help target
help:
	@echo "Legacy32 Contemplation Period Makefile"
	@echo "======================================="
	@echo ""
	@echo "Targets:"
	@echo "  all      - Build the contemplation binary (default)"
	@echo "  clean    - Remove build artifacts"
	@echo "  install  - Install to /usr/local/bin (requires root)"
	@echo "  run      - Build and run the contemplation period"
	@echo "  help     - Show this help message"

.PHONY: all clean install run help
