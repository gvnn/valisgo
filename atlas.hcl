data "external_schema" "gorm_postgres" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/domain",
    "--dialect", "postgres",
  ]
}

env "postgres" {
  src = data.external_schema.gorm_postgres.url
  dev = "docker://postgres/18/dev?search_path=public"
  migration {
    dir = "file://migrations/postgres"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
