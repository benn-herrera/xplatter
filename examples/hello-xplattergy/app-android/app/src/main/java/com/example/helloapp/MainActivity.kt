package com.example.helloapp

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import hello.xplattergy.HelloXplattergy

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { HelloScreen() }
    }
}

@Composable
fun HelloScreen() {
    var nameInput by remember { mutableStateOf("xplattergy") }
    var greetingOutput by remember { mutableStateOf("") }
    val greeter = remember {
        try {
            HelloXplattergy.createGreeter()
        } catch (e: Exception) {
            null
        }
    }

    Column(
        modifier = Modifier.padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(20.dp)
    ) {
        TextField(
            value = nameInput,
            onValueChange = { nameInput = it },
            label = { Text("Enter a name") }
        )
        Button(onClick = {
            greetingOutput = try {
                greeter?.sayHello(nameInput)?.message ?: "No greeter"
            } catch (e: Exception) {
                "Error: ${e.message}"
            }
        }) {
            Text("Greet")
        }
        Text(greetingOutput)
    }
}
