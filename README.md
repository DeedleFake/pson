pson
====

This is an experimental implementation of progressive JSON similar to what is described in https://youtu.be/MaMQLNBZz64. The idea is based on progressive images where instead of loading the entire image from the top to the bottom over the network, a lower quality version of the entire image is sent first and then the details are filled in afterwards. This works by sending special JSON tags that correspond to data that is sent after the entire original object is sent, and this process can be repeated recursively, allowing data to not only be partially handled before all of it has been loaded, but also allowing cheaper to calculate data to be sent by the server before more expensive data is even ready to send at all.
