-- Modify "repositories" table
ALTER TABLE "repositories" ADD COLUMN "type" character varying(50) NOT NULL DEFAULT 'local', ADD COLUMN "upstream_url" character varying(255) NULL;
-- Create "virtual_repo_members" table
CREATE TABLE "virtual_repo_members" (
  "virtual_repo_id" bigint NOT NULL,
  "member_repo_id" bigint NOT NULL,
  "priority" bigint NOT NULL DEFAULT 0,
  PRIMARY KEY ("virtual_repo_id", "member_repo_id"),
  CONSTRAINT "fk_repositories_virtual_members" FOREIGN KEY ("virtual_repo_id") REFERENCES "repositories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
