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
