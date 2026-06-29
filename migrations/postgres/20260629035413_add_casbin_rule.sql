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
