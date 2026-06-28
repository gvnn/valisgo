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

data "external_schema" "gorm_sqlite" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/domain",
    "--dialect", "sqlite",
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

env "sqlite" {
  src = data.external_schema.gorm_sqlite.url
  dev = "sqlite://file?mode=memory&_fk=1"
  migration {
    dir = "file://migrations/sqlite"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
