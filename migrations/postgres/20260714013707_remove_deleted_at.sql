-- Modify "package_files" table
ALTER TABLE "package_files" DROP COLUMN "deleted_at";
-- Modify "packages" table
ALTER TABLE "packages" DROP COLUMN "deleted_at";
-- Modify "registries" table
ALTER TABLE "registries" DROP COLUMN "deleted_at";
-- Modify "repositories" table
ALTER TABLE "repositories" DROP COLUMN "deleted_at";
