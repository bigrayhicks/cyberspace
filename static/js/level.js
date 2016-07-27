var Level = (function () {

    "use strict";

    var engine;

    var scene;
    // contains the last unix timestamp received from the websocket
    var timestamp;

    var serverTick;
    var objects = {};

    var materials = {};
    var models = [];

    var canvas;

    var sequence = 0;

    var mainText =
        new BABYLON.Text2D("cyberspace", {
            id: "text",
            marginAlignment: "h: left, v:center",
            fontName: "20pt Arial"
        });

    var setupModels = function (scene) {
        // Material selection
        materials.blue = new BABYLON.StandardMaterial("texture1", scene);
        materials.blue.diffuseColor = new BABYLON.Color3(0.0, 0.0, 0.4);

        materials.gray = new BABYLON.StandardMaterial("texture1", scene);
        materials.gray.diffuseColor = new BABYLON.Color3(0.2, 0.9, 1);
        materials.gray.diffuseTexture = new BABYLON.Texture("/assets/square_gray.jpg", scene);

        materials.yellow = new BABYLON.StandardMaterial("yellow", scene);
        materials.yellow.diffuseColor = new BABYLON.Color3(0.9, 0.8, 0.7);
        materials.yellow.diffuseTexture = new BABYLON.Texture("/assets/square_running.jpeg", scene);

        models[0] = BABYLON.Mesh.CreateBox("box", 1.0, scene, false, BABYLON.Mesh.DEFAULTSIDE);
        models[0].scaling = new BABYLON.Vector3(10, 10, 10);
        models[0].material = materials.gray;
        models[0].isVisible = false;

        models[1] = BABYLON.Mesh.CreateBox("box", 1.0, scene, false, BABYLON.Mesh.DEFAULTSIDE);
        models[1].scaling = new BABYLON.Vector3(30, 30, 30);
        models[1].isVisible = false;
        models[1].material = materials.yellow;
    };

    // Over/Out
    var onClick = function (mesh) {
        var action = new BABYLON.ExecuteCodeAction(BABYLON.ActionManager.OnPickTrigger, function (evt) {
            var xhr = new XMLHttpRequest();
            xhr.open('POST', encodeURI('/monitor'));
            xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
            xhr.onload = function () {
                if (xhr.status !== 200) {
                    alert('Request failed.  Returned status of ' + xhr.status);
                } else {
                    var info = JSON.parse(xhr.responseText);
                    var text = info.Name + " - " + info.InstanceType + " " + info.InstanceID + "\n";
                    text += "CPU utilisation: " + info.CPUUtilization + "%\n";
                    if(info.HasCredits) {
                        text += "CPU credits: " + info.CPUCreditBalance;
                    }
                    Level.changeText(text);
                }
            };
            xhr.send(encodeURI('id=' + evt.source.id));
        });
        mesh.actionManager.registerAction(action);
    };

    var updateScene = function (updates) {

        for (var id in updates) {
            if (!updates.hasOwnProperty(id)) {
                continue;
            }

            // entity needs to be created
            if (typeof objects[id] === 'undefined') {
                objects[id] = models[1].clone(id);
                objects[id].id = id;
                objects[id].isVisible = true;
                objects[id].actionManager = new BABYLON.ActionManager(scene);
                objects[id].material = new BABYLON.StandardMaterial(id, scene);
                objects[id].material.diffuseColor = new BABYLON.Color3(0.9, 0.8, 0.7);
                objects[id].material.specularColor = new BABYLON.Color3(0.2, 0.2, 0.2);
                objects[id].material.diffuseTexture = new BABYLON.Texture("/assets/square_running.jpeg", scene);
                onClick(objects[id]);
            }


            objects[id].position = updates[id].position;
            objects[id].rotationQuaternion = new BABYLON.Quaternion(updates[id].orientation[1], updates[id].orientation[2], updates[id].orientation[3], updates[id].orientation[0]);
            objects[id].scaling = updates[id].scale;

            if (updates[id].health > 0.99) {
                objects[id].material.diffuseColor = new BABYLON.Color3(0.2, 0.2, 0.2);
            } else if (updates[id].health > 0.98) {
                objects[id].material.diffuseColor = new BABYLON.Color3(0.3, 0.2, 0.1);
            } else if (updates[id].health > 0.66) {
                objects[id].material.diffuseColor = new BABYLON.Color3(0.9, 0.8, 0.7);
            } else if (updates[id].health > 0.10) {
                objects[id].material.diffuseColor = new BABYLON.Color3(0.9, 0.5, 0.3);

            } else {
                objects[id].material.diffuseColor = new BABYLON.Color3(0.9, 0.0, 0.0);
            }
            //} else if(updates[id].health > 0.5) {
            //    objects[id].material.diffuseColor = new BABYLON.Color3(0.9, 0.5, 0.4);
            //} else if(updates[id].health > 0.25) {
            //    objects[id].material.diffuseColor = new BABYLON.Color3(0.9, 0.0, 0.0);

        }
    };

    var entityInfo = function (buf) {
        var cmd = buf.readUint8();
        console.log(cmd);
        var objectId = buf.readFloat32();
        console.log(objectId);
        Level.changeText(cmd + " " + objectId);
    };

    var entityUpdate = function (buf) {
        var objectId;
        var updates = [];
        while (!buf.isEof()) {
            var cmd = buf.readUint8();
            switch (cmd) {
                // INST_ENTITY_ID - we are switching the object we wish to update
                case 1:
                    objectId = buf.readFloat32();
                    updates[objectId] = new Update();
                    break;
                // INST_SET_POSITION
                case 2:
                    var pos = {
                        x: buf.readFloat32(),
                        y: buf.readFloat32(),
                        z: buf.readFloat32()
                    };
                    updates[objectId].position = pos;
                    break;
                // INST_SET_ROTATION
                case 3:
                    updates[objectId].orientation = [];
                    updates[objectId].orientation[0] = buf.readFloat32();
                    updates[objectId].orientation[1] = buf.readFloat32();
                    updates[objectId].orientation[2] = buf.readFloat32();
                    updates[objectId].orientation[3] = buf.readFloat32();
                    break;
                // INST_SET_MODEL
                case 4:
                    updates[objectId].model = buf.readFloat32();
                    break;
                // INST_SET_SCALE
                case 5:
                    updates[objectId].scale = {
                        x: buf.readFloat32(),
                        y: buf.readFloat32(),
                        z: buf.readFloat32()
                    };
                    break;
                case 6:
                    updates[objectId].health = buf.readFloat32();
                    break;
            }
        }

        updateScene(updates);
    };

    return {

        init: function (e, s) {
            engine = e;
            scene = s;


            setupModels(s);

            canvas = new BABYLON.ScreenSpaceCanvas2D(scene, {
                id: "ScreenCanvas",
                size: new BABYLON.Size(600, 100),
                children: [
                    mainText
                ]
            });

        },

        changeText: function (text) {
            mainText.text = text
        },

        update: function (buf) {

            timestamp = buf.readFloat64();
            var msgType = buf.readUint8();
            serverTick = buf.readFloat32();

            switch (msgType) {
                case 1:
                    entityUpdate(buf);
                    break;
                case 2:
                    entityInfo(buf);
                    break;
                default:
                    console.log("Not sure what to do with message type " + msgType);
                    break;
            }

        }
    }
})();
