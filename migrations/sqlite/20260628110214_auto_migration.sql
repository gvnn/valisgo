-- Create "registries" table
CREATE TABLE `registries` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `name` text NOT NULL,
  `format` varchar NULL DEFAULT 'file'
);
-- Create index "idx_registries_name" to table: "registries"
CREATE UNIQUE INDEX `idx_registries_name` ON `registries` (`name`);
-- Create index "idx_registries_deleted_at" to table: "registries"
CREATE INDEX `idx_registries_deleted_at` ON `registries` (`deleted_at`);
