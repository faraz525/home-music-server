#!/bin/bash
# Quick script to apply database optimizations locally

set -e

echo "🗄️  Applying Database Optimizations..."
echo ""

# Step 1: Backup current database (optional, but safe)
if [ -f "data/cratedrop/db/cratedrop.sqlite" ]; then
    echo "📦 Backing up current database..."
    BACKUP_FILE="data/cratedrop/db/backup_$(date +%Y%m%d_%H%M%S).sqlite"
    cp data/cratedrop/db/cratedrop.sqlite "$BACKUP_FILE"
    echo "   ✅ Backup saved to: $BACKUP_FILE"
    echo ""
fi

# Step 2: Remove old database files
echo "🗑️  Removing old database files..."
rm -f data/cratedrop/db/cratedrop.sqlite*
echo "   ✅ Old database removed"
echo ""

# Step 3: Database will be recreated on next run with new schema
echo "✨ Done! New optimized schema will be created on next run."
echo ""
echo "📝 Next steps:"
echo "   1. Run: cd backend && ./run-local.sh"
echo "   2. Sign up a new user"
echo "   3. Upload tracks and test the speed!"
echo ""
echo "🚀 Your database is now optimized for Raspberry Pi!"

