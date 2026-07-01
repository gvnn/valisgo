-- Modify "virtual_repo_members" table
ALTER TABLE "virtual_repo_members" ADD CONSTRAINT "fk_virtual_repo_members_member_repo" FOREIGN KEY ("member_repo_id") REFERENCES "repositories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
