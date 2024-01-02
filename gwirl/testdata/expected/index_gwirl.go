package html

import "github.com/gamebox/gwirl/gwirl-example/model"
import (
    "github.com/gamebox/gwirl"
)


func Index(participants []model.Participant) string {
    sb_ := gwirl.TemplateBuilder{}

    sb_.WriteString(`
`)

    var transclusion__5__1 string
    {
        sb_ := gwirl.TemplateBuilder{}
        sb_.WriteString(`
    <main class="bg-gray-500">
        `)

        sb_.WriteString(ManageParticipants(participants))

        sb_.WriteString(`
        `)

        sb_.WriteString(Fun("This is a message"))

        sb_.WriteString(`
    </main>
`)

        transclusion__5__1 = sb_.String()
    }
    sb_.WriteString(Base(nil, "Gwirl HTML Example", "/", transclusion__5__1))


    sb_.WriteString(`
`)

    return sb_.String()
}
