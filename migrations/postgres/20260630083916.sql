-- Create "casbin_rule" table
CREATE TABLE "casbin_rule" (
  "id" bigserial NOT NULL,
  "ptype" character varying(512) NULL,
  "v0" character varying(512) NULL,
  "v1" character varying(512) NULL,
  "v2" character varying(512) NULL,
  "v3" character varying(512) NULL,
  "v4" character varying(512) NULL,
  "v5" character varying(512) NULL,
  PRIMARY KEY ("id")
);
-- Create index "unique_index" to table: "casbin_rule"
CREATE UNIQUE INDEX "unique_index" ON "casbin_rule" ("ptype", "v0", "v1", "v2", "v3", "v4", "v5");
-- Create "registries" table
CREATE TABLE "registries" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(255) NOT NULL,
  "format" character varying(255) NULL DEFAULT 'file',
  PRIMARY KEY ("id")
);
-- Create index "idx_registries_deleted_at" to table: "registries"
CREATE INDEX "idx_registries_deleted_at" ON "registries" ("deleted_at");
-- Create index "idx_registries_name" to table: "registries"
CREATE UNIQUE INDEX "idx_registries_name" ON "registries" ("name");
-- Create "repositories" table
CREATE TABLE "repositories" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(255) NOT NULL,
  "registry_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_repositories_registry" FOREIGN KEY ("registry_id") REFERENCES "registries" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_name_registry" to table: "repositories"
CREATE UNIQUE INDEX "idx_name_registry" ON "repositories" ("name", "registry_id");
-- Create index "idx_repositories_deleted_at" to table: "repositories"
CREATE INDEX "idx_repositories_deleted_at" ON "repositories" ("deleted_at");
-- Create "packages" table
CREATE TABLE "packages" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(255) NOT NULL,
  "normalized_name" character varying(255) NOT NULL,
  "repository_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_packages_repository" FOREIGN KEY ("repository_id") REFERENCES "repositories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_packages_deleted_at" to table: "packages"
CREATE INDEX "idx_packages_deleted_at" ON "packages" ("deleted_at");
-- Create index "idx_packages_normalized_name" to table: "packages"
CREATE INDEX "idx_packages_normalized_name" ON "packages" ("normalized_name");
-- Create index "idx_repository_normalized_name" to table: "packages"
CREATE UNIQUE INDEX "idx_repository_normalized_name" ON "packages" ("repository_id");
-- Create "package_files" table
CREATE TABLE "package_files" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "package_id" bigint NOT NULL,
  "version" character varying(255) NOT NULL,
  "filename" character varying(255) NOT NULL,
  "hash" character varying(255) NOT NULL,
  "size" bigint NOT NULL,
  "blob_key" character varying(255) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_packages_files" FOREIGN KEY ("package_id") REFERENCES "packages" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_package_filename" to table: "package_files"
CREATE UNIQUE INDEX "idx_package_filename" ON "package_files" ("package_id", "filename");
-- Create index "idx_package_files_deleted_at" to table: "package_files"
CREATE INDEX "idx_package_files_deleted_at" ON "package_files" ("deleted_at");
