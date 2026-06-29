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
-- Create index "idx_repositories_deleted_at" to table: "repositories"
CREATE INDEX "idx_repositories_deleted_at" ON "repositories" ("deleted_at");
-- Create index "idx_repositories_name" to table: "repositories"
CREATE UNIQUE INDEX "idx_repositories_name" ON "repositories" ("name");
