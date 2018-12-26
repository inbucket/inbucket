module Page.Home exposing (Model, Msg, init, update, view)

import Api
import Data.Session as Session exposing (Session)
import Html exposing (..)
import Html.Attributes exposing (..)
import Http
import HttpUtil
import Json.Encode as Encode
import Ports



-- MODEL --


type alias Model =
    { session : Session
    , greeting : String
    }


init : Session -> ( Model, Cmd Msg )
init session =
    ( Model session "", Api.getGreeting GreetingLoaded )



-- UPDATE --


type Msg
    = GreetingLoaded (Result HttpUtil.Error String)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GreetingLoaded (Ok greeting) ->
            ( { model | greeting = greeting }, Cmd.none )

        GreetingLoaded (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            )



-- VIEW --


view : Model -> { title : String, modal : Maybe (Html msg), content : List (Html Msg) }
view model =
    { title = "Inbucket"
    , modal = Nothing
    , content =
        [ Html.node "rendered-html"
            [ class "greeting"
            , property "content" (Encode.string model.greeting)
            ]
            []
        ]
    }
