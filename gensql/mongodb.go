package gensql

type MongoDBDriver struct{}

func (d *MongoDBDriver) Name() string { return "mongodb" }

func (d *MongoDBDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	// Apply defaults and bounds for MongoDB-specific options
	sampleSize := opts.SampleSize
	if sampleSize <= 0 {
		sampleSize = 100
	} else if sampleSize > 10000 {
		sampleSize = 10000 // Cap at 10k to prevent memory issues
	}
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	} else if maxDepth > 10 {
		maxDepth = 10 // Cap depth to prevent excessive recursion
	}

	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   false, // MongoDB uses direct script
		DBName:       opts.DBName,
		SPName:       opts.SPName,
		SampleSize:   sampleSize,
		MaxDepth:     maxDepth,
	}

	script, err := ExecuteTemplate("mongodb", vars)
	if err != nil {
		return "", err
	}

	return script, nil
}
