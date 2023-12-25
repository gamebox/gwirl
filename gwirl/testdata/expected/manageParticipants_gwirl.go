package html

import "github.com/gamebox/gwirl/gwirl-example/model"
import (
    "strings"

    "github.com/gamebox/gwirl"
)


func ManageParticipants(participants []model.Participant) string {
    sb_ := strings.Builder{}

    sb_.WriteString(`
<div class="title-container">
    <h1>Participant Manager</h1>
    <a class="action" target="_blank" href="/cpc/">New</a>
</div>

<table class="list" cellspacing="0" cellpadding="0">
    <thead>
        <tr>
            <th>First Name</th>
            <th>Last Name</th>
            <th>Email</th>
            <th>Actions</th>
        </tr>
    </thead>
    <tbody>
        `)

    if  len(participants) == 0  {
        sb_.WriteString(`
            <tr>
                <td colspan="4">No participants</td>
            </tr>
        `)

    }

    sb_.WriteString(`
        `)

    for  _, participant := range participants  {
        sb_.WriteString(`
            <tr>
                <td>`)

        gwirl.WriteEscapedHTML(&sb_, participant.FirstName)

        sb_.WriteString(`</td>
                <td>`)

        gwirl.WriteEscapedHTML(&sb_, participant.LastName)

        sb_.WriteString(`</td>
                <td>`)

        gwirl.WriteEscapedHTML(&sb_, participant.Email)

        sb_.WriteString(`</td>
                <td>
                    <a href="#" class="action">Delete</a>
                    <a href="/client/participant/`)

        gwirl.WriteEscapedHTML(&sb_, participant.Id)

        sb_.WriteString(`" class="action">Edit</a>
                </td>
            </tr>
        `)

    }

    sb_.WriteString(`
    </tbody>
</table>
`)

    return sb_.String()
}
