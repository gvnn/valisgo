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
