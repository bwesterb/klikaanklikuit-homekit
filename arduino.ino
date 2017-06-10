const int pin_radiotx = 8;


void setup() {
    Serial.begin(115200);
}

void loop() {
    Serial.println("?");

    while(!Serial.available());
    char cmd = Serial.read();

    if (cmd == 'R') {
        radio_transmit();
    } else {
        Serial.println("C");
    }
}

void radio_transmit()
{
    int n_pulses, n_switches;
    int* switches = NULL;

    while(!Serial.available());
    n_pulses = Serial.readStringUntil('\n').toInt();
    n_switches = n_pulses * 2 - 1;

    switches = (int*)malloc(sizeof(int) * n_switches);

    if (switches == NULL) {
        Serial.println("M");
        return;
    }

    for (int i = 0; i < n_switches; i++) {
        while(!Serial.available());
        switches[i] = Serial.readStringUntil('\n').toInt();
    }

    pinMode(pin_radiotx, OUTPUT);

    for (int i = 0; i < n_switches; i++) {
        digitalWrite(pin_radiotx, i % 2 ? LOW : HIGH);
        delayMicroseconds(switches[i]);
    }

    digitalWrite(pin_radiotx, LOW);

    free(switches);

    Serial.println("!");
}
