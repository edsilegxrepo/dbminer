package gensql

type MariaDBDriver struct{}

func (d *MariaDBDriver) Name() string { return "mariadb" }

func (d *MariaDBDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   !opts.Direct,
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("mariadb", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "mariadb"), nil
}
