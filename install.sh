#!/bin/bash

# Define the installation directory and subdirectories
HOME_DIR="/opt/job-manager"
BIN_DIR="$HOME_DIR/bin"
LOG_DIR="$HOME_DIR/logs"
CONFIG_DIR="$HOME_DIR/config"
RESOURCES_DIR="$HOME_DIR/resources"
EXEC_DIR="./"  # Path where your executables are located

# Check for root privileges (since writing to /opt usually requires sudo)
if [ "$(id -u)" != "0" ]; then
  echo "This script must be run as root (use sudo)."
  exit 1
fi

echo "======================================================================="
echo "                         Installing JobManager                         "
echo "======================================================================="
# Create the installation directory and subdirectories if they do not exist
echo "======================================================================="
echo "Creating installation directories..."
echo "======================================================================="
dirs=("$HOME_DIR" "$BIN_DIR" "$LOG_DIR" "$CONFIG_DIR" "$RESOURCES_DIR")
for dir in "${dirs[@]}"; do
  if [ ! -d "$dir" ]; then  # Check if the directory doesn't exist
      echo "Creating directory: $dir"
      mkdir -p "$dir"  # Create the directory, including parent directories
  else
      echo "Directory already exists: $dir"
  fi
done

# Copy all executable files from the current directory to the install directory
echo "======================================================================="
echo "Copying executables to $HOME_DIR..."
echo "======================================================================="
EXECUTABLES=("jobManager")
for EXECUTABLE in "${EXECUTABLES[@]}"; do
  echo "Copying the executable : $EXECUTABLE to the directory $BIN_DIR"
  cp "$EXEC_DIR"/"$EXECUTABLE" "$BIN_DIR/"
done

# Set the correct permissions for the installation directory and executables
echo "======================================================================="
echo "Setting permissions..."
chmod -R 755 "$HOME_DIR"   # Grant read, write, and execute permissions for the owner, read and execute for others
chmod -R +x "$HOME_DIR"/*  # Ensure that executables have execute permissions

# Provide feedback to the user
echo "======================================================================="
echo "Home directory: $HOME_DIR"
echo "Bin directory: $BIN_DIR"
echo "Logs directory: $LOG_DIR"
echo "Resources directory: $RESOURCES_DIR"
echo "Config directory: $CONFIG_DIR"
echo "======================================================================="
echo "                        Installation complete!                         "
echo "======================================================================="