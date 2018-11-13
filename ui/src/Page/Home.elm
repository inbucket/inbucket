module Page.Home exposing (Model, Msg, init, update, view)

import Data.Session as Session exposing (Session)
import Html exposing (..)
import Html.Attributes exposing (..)
import Http
import HttpUtil
import Json.Encode as Encode
import Ports


-- MODEL --


type alias Model =
    { greeting : String }


init : ( Model, Cmd Msg )
init =
    ( Model ""
    , Cmd.batch
        [ Ports.windowTitle "Inbucket"
        , cmdGreeting
        ]
    )


cmdGreeting : Cmd Msg
cmdGreeting =
    Http.send GreetingResult <|
        Http.getString "/serve/greeting"



-- UPDATE --


type Msg
    = GreetingResult (Result Http.Error String)


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        GreetingResult (Ok greeting) ->
            ( Model greeting, Cmd.none, Session.none )

        GreetingResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )



-- VIEW --


view : Session -> Model -> Html Msg
view session model =
    div [ id "page" ]
        [ div
            [ class "greeting"
            , property "innerHTML" (Encode.string model.greeting)
            ]
            []
        ]
