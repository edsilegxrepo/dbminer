package gensql

type MySQL57Driver struct{}

func (d *MySQL57Driver) Name() string { return "mysql57" }

func (d *MySQL57Driver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   !opts.Direct,
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("mysql57", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "mysql"), nil
}
