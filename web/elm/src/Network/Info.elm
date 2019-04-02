module Network.Info exposing (fetch)

import Concourse
import Http
import Task exposing (Task)


fetch : Task Http.Error Concourse.ConcourseVersion
fetch =
    Http.toTask <| Http.get "/api/v1/info" Concourse.decodeInfo
