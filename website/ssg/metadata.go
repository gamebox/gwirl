package ssg

type Metadata struct {
    Title string
    Description string
    Author string
    Tags []string
    Slug string
}

func NewMetadata(raw map[string]interface{}) Metadata {
    meta := Metadata{}
    if title, ok := raw["title"].(string); ok {
        meta.Title = title
    }
    if description, ok := raw["description"].(string); ok {
        meta.Description = description
    }
    if author, ok := raw["author"].(string); ok {
        meta.Author = author
    }
    if slug, ok := raw["slug"].(string); ok {
        meta.Slug = slug
    }
    if path, ok := raw["_filepath"].(string); ok {
        meta.Slug = path
    }
    if tagsRaw, ok := raw["tags"].([]interface{}); ok && meta.Slug == "" {
        tags := make([]string, 0, len(tagsRaw))
        for _, tagRaw := range tagsRaw {
            if tag, ok := tagRaw.(string); ok {
                tags = append(tags, tag)
            }
        }
        meta.Tags = tags
    }

    return meta
}

