package html

import "github.com/gamebox/gwirl/gwirl-example/model"
import (
    "github.com/gamebox/gwirl"
)


func ManageParticipants(participants []model.Participant) string {
    sb_ := gwirl.TemplateBuilder{}

    sb_.WriteString(`
<div class="md:flex md:items-center md:justify-between">
    <h1 class="text-2xl font-bold leading-7 text-white sm:truncate sm:text-3xl sm:tracking-tight">Participant Manager</h1>
    <a class="ml-3 inline-flex items-center rounded-md bg-green-600 px-3 py-2 text-sm font-semibold text-white shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-green-500 hover:bg-green-500" target="_blank" href="/cpc/">New</a>
</div>

<table class="w-full" cellspacing="0" cellpadding="0">
    <thead>
        <tr>
            <th class="bg-gray-500">First Name</th>
            <th class="bg-gray-500">Last Name</th>
            <th class="bg-gray-500">Email</th>
            <th class="bg-gray-500">Actions</th>
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

        sb_.WriteString(participant.Id)

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
