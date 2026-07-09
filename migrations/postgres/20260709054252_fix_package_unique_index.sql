-- Drop index "idx_packages_normalized_name" from table: "packages"
DROP INDEX "idx_packages_normalized_name";
-- Drop index "idx_repository_normalized_name" from table: "packages"
DROP INDEX "idx_repository_normalized_name";
-- Create index "idx_repository_normalized_name" to table: "packages"
CREATE UNIQUE INDEX "idx_repository_normalized_name" ON "packages" ("normalized_name", "repository_id");
