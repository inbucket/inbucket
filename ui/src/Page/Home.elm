module Page.Home exposing (Model, Msg, init, update, view)

import Data.Session exposing (Session)
import Effect exposing (Effect)
import Html exposing (Html)
import Html.Attributes exposing (class, property)
import HttpUtil
import Json.Encode as Encode



-- MODEL --


type alias Model =
    { session : Session
    , greeting : String
    }


init : Session -> ( Model, Effect Msg )
init session =
    ( Model session "", Effect.getGreeting GreetingLoaded )



-- UPDATE --


type Msg
    = GreetingLoaded (Result HttpUtil.Error String)


update : Msg -> Model -> ( Model, Effect Msg )
update msg model =
    case msg of
        GreetingLoaded (Ok greeting) ->
            ( { model | greeting = greeting }, Effect.none )

        GreetingLoaded (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )



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
