#!/bin/bash
# Generate password hash for GC_APPROVAL_PASSWORD_HASH
# Run this script, enter your password, then copy the hash to .env

echo -n "Enter password to hash: "
read -s password
echo ""

hash=$(echo -n "$password" | sha256sum | cut -d' ' -f1)

echo ""
echo "Add this line to your .env file:"
echo ""
echo "GC_APPROVAL_PASSWORD_HASH=$hash"
echo ""
echo "The monitor will verify you know the password by hashing your input"
echo "and comparing to this stored hash. The plaintext is never stored."
