package gensql

type MySQLDriver struct{}

func (d *MySQLDriver) Name() string { return "mysql" }

func (d *MySQLDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   !opts.Direct,
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("mysql", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "mysql"), nil
}
