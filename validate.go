package main

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/twmb/franz-go/pkg/sr"
)

func validateS(a Source, b Source) {
	ra, aok := a.(*RestSource)
	if aok {
		bs, err := b.GetState()
		if err != nil {
			panic(fmt.Errorf("No state"))
		}
		validateR(ra, bs)
	}
	rb, bok := b.(*RestSource)
	if bok {
		as, err := a.GetState()
		if err != nil {
			panic(fmt.Errorf("No state"))
		}
		validateR(rb, as)
	}
}

func validateR(rs *RestSource, s *State) {
	success := 0
	normalized := 0
	not_found := 0
	failed := 0
	for _, subjectSchema := range s.SubjectSchemas {
		path := url.PathEscape(subjectSchema.Subject)
		_, err := rs.LookupSchema(subjectSchema.Subject, subjectSchema.Schema, sr.ShowDeleted)
		if err == nil {
			success += 1
			fmt.Printf("%vsubjects/%v/versions/%v found\n", rs.GetURL(), path, subjectSchema.Version)
		}
		_, err = rs.LookupSchema(subjectSchema.Subject, subjectSchema.Schema, sr.ShowDeleted, sr.Normalize)
		if err == nil {
			normalized += 1
			fmt.Printf("%vsubjects/%v/versions/%v found normalized, deleted\n", rs.GetURL(), path, subjectSchema.Version)
			continue
		}
		sv, err := rs.SchemaByVersion(subjectSchema.Subject, subjectSchema.Version)
		if err != nil {
			failed += 1
			fmt.Printf("%vsubjects/%v/versions/%v not found:\n%v\n", rs.GetURL(), path, subjectSchema.Version, subjectSchema.Schema)
		} else {
			not_found += 1

			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(subjectSchema.Schema.Schema),
				B:        difflib.SplitLines(sv.Schema.Schema),
				FromFile: path,
				ToFile:   sv.Subject,
				Context:  3,
			}
			result, _ := difflib.GetUnifiedDiffString(diff)
			fmt.Printf("%vsubjects/%v/versions/%v not found:\n%v\n%v\n", rs.GetURL(), path, subjectSchema.Version, result, subjectSchema.Schema.References)
		}
	}
	fmt.Printf("RESULTS: success: %v, normalized: %v, not_found: %v, failed: %v\n", success, normalized, not_found, failed)
}

func validate(a *State, b *State) {

	// Build Lookup Tables

	aSubjectSchemas := make(map[sr.SubjectVersion]*sr.SubjectSchema)
	for _, subjectSchema := range a.SubjectSchemas {
		ref := getReference(subjectSchema)
		aSubjectSchemas[ref] = &subjectSchema
	}

	aCompatibilityResults := make(map[string]*sr.CompatibilityResult)
	for _, compatibilityResult := range a.CompatibilityResults {
		aCompatibilityResults[compatibilityResult.Subject] = &compatibilityResult
	}

	aSoftDeletions := make(map[sr.SubjectVersion]bool)
	for _, reference := range a.SoftDeletions {
		aSoftDeletions[reference] = true
	}

	bSubjectSchemas := make(map[sr.SubjectVersion]*sr.SubjectSchema)
	for _, subjectSchema := range b.SubjectSchemas {
		ref := getReference(subjectSchema)
		bSubjectSchemas[ref] = &subjectSchema
	}

	bCompatibilityResults := make(map[string]*sr.CompatibilityResult)
	for _, compatibilityResult := range b.CompatibilityResults {
		bCompatibilityResults[compatibilityResult.Subject] = &compatibilityResult
	}

	bSoftDeletions := make(map[sr.SubjectVersion]bool)
	for _, reference := range b.SoftDeletions {
		bSoftDeletions[reference] = true
	}

	// Compare schemaSubjects

	for _, aSubjectSchema := range a.SubjectSchemas {
		bSubjectSchema, ok := bSubjectSchemas[getReference(aSubjectSchema)]
		if !ok {
			panic(fmt.Errorf("subject %v version %v not found in right", aSubjectSchema.Subject, aSubjectSchema.Version))
		} else {
			if aSubjectSchema.ID != bSubjectSchema.ID {
				panic(fmt.Errorf("subject %v version %v schema IDs don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.ID, bSubjectSchema.ID))
			}
			if aSubjectSchema.ID != bSubjectSchema.ID {
				panic(fmt.Errorf("subject %v version %v schemas don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.Schema, bSubjectSchema.Schema))
			}
			if aSubjectSchema.Type != bSubjectSchema.Type {
				panic(fmt.Errorf("subject %v version %v schema types don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.Type, bSubjectSchema.Type))
			}
			if !reflect.DeepEqual(aSubjectSchema.References, bSubjectSchema.References) {
				panic(fmt.Errorf("subject %v version %v references don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.References, bSubjectSchema.References))
			}
		}
	}

	for _, bSubjectSchema := range b.SubjectSchemas {
		aSubjectSchema, ok := aSubjectSchemas[getReference(bSubjectSchema)]
		if !ok {
			panic(fmt.Errorf("subject %v version %v not found in left", bSubjectSchema.Subject, bSubjectSchema.Version))
		} else {
			if aSubjectSchema.ID != bSubjectSchema.ID {
				panic(fmt.Errorf("subject %v version %v schema IDs don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.ID, bSubjectSchema.ID))
			}
			if aSubjectSchema.ID != bSubjectSchema.ID {
				panic(fmt.Errorf("subject %v version %v schemas don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.Schema, bSubjectSchema.Schema))
			}
			if aSubjectSchema.Type != bSubjectSchema.Type {
				panic(fmt.Errorf("subject %v version %v schema types don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.Type, bSubjectSchema.Type))
			}
			if !reflect.DeepEqual(aSubjectSchema.References, bSubjectSchema.References) {
				panic(fmt.Errorf("subject %v version %v references don't match: %v vs %v", aSubjectSchema.Subject, aSubjectSchema.Version, aSubjectSchema.References, bSubjectSchema.References))
			}
		}
	}
}
